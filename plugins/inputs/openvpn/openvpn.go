package openvpn

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
)

const (
	description  = "Gather status from the OpenVPN management interface"
	sampleConfig = `
  ## Address of the management interface
  ##   ex: address = "tcp://example.org:1234"
  address = "unix:///var/run/openvpn.sock"

  ## Password to the management interface
  # password = ""

  ## Timeout for connecting and getting status.
  # timeout = "5s"
`
)

const (
	defaultTimeout = 5 * time.Second
)

type OpenVPN struct {
	Address  string          `toml:"address"`
	Password string          `toml:"password"`
	Timeout  config.Duration `toml:"timeout"`

	conn   net.Conn
	scheme string
	addr   string
}

func (o *OpenVPN) SampleConfig() string {
	return sampleConfig
}

func (o *OpenVPN) Description() string {
	return description
}

func (o *OpenVPN) Init() error {
	if o.Timeout < config.Duration(time.Second) {
		o.Timeout = config.Duration(defaultTimeout)
	}

	parts, err := url.Parse(o.Address)
	if err != nil {
		return err
	}

	o.scheme = parts.Scheme

	switch parts.Scheme {
	case "tcp", "tcp4", "tcp6":
		o.addr = parts.Host
	case "unix":
		o.addr = parts.Path
	default:
		return fmt.Errorf("unsupported scheme %q", parts.Scheme)
	}
	return nil
}

func (o *OpenVPN) Start(telegraf.Accumulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.Timeout))
	defer cancel()

	var err error

	var dialer net.Dialer
	o.conn, err = dialer.DialContext(ctx, o.scheme, o.addr)
	if err != nil {
		return err
	}

	if deadline, ok := ctx.Deadline(); ok {
		o.conn.SetDeadline(deadline)
	}

	_, err = o.conn.Write([]byte("pid\n"))
	if err != nil {
		return err
	}

	r := bufio.NewReader(o.conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		return err
	}

	if len(line) != 0 && bytes.HasPrefix(line, []byte("SUCCESS:")) {
		return nil
	}

	return fmt.Errorf("unexpected response %q", line)
}

func (o *OpenVPN) Gather(acc telegraf.Accumulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.Timeout))
	defer cancel()

	var err error
	if o.conn == nil {
		var dialer net.Dialer
		o.conn, err = dialer.DialContext(ctx, o.scheme, o.addr)
		if err != nil {
			return err
		}
	}

	err = o.getStatus(ctx, acc)
	if err != nil {
		o.conn = nil
		return err
	}

	return nil
}

type column struct {
	header    string
	key       string
	valueFunc func(string, string, telegraf.Metric)
}

func ignoreField(string, string, telegraf.Metric) {
}

func addStringField(key string, value string, m telegraf.Metric) {
	if value == "" || value == "UNDEF" {
		return
	}
	m.AddField(key, value)
}

func addTag(key string, value string, m telegraf.Metric) {
	if value == "" || value == "UNDEF" {
		return
	}
	m.AddTag(key, value)
}

func addUnsignedField(key string, value string, m telegraf.Metric) {
	u, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return
	}
	m.AddField(key, u)
}

func addTimeField(key string, value string, m telegraf.Metric) {
	unix, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return
	}
	m.AddField(key, unix*100000000)
}

var clientListFields = []column{
	{"CLIENT_LIST", "", ignoreField},
	{"Common Name", "common_name", addTag},
	{"Read Address", "real_address", addStringField},
	{"Virtual Address", "virtual_address", addStringField},
	{"Virtual IPv6 Address", "virtual_ipv6_address", addStringField},
	{"Bytes Recieved", "bytes_received", addUnsignedField},
	{"Bytes Sent", "bytes_sent", addUnsignedField},
	{"Connected Since", "connected_since", ignoreField},
	{"Connected Since (time_t)", "connected_since", addTimeField},
	{"Username", "username", addStringField},
	{"Client ID", "client_id", addStringField},
	{"Peer ID", "peer_id", addTag},
}

func (o *OpenVPN) getStatus(ctx context.Context, acc telegraf.Accumulator) error {
	if deadline, ok := ctx.Deadline(); ok {
		o.conn.SetDeadline(deadline)
	}

	_, err := o.conn.Write([]byte("status 2\n"))
	if err != nil {
		return err
	}

	r := csv.NewReader(o.conn)
	r.FieldsPerRecord = -1

	for {
		record, err := r.Read()
		// fmt.Printf("debug %q\n", record)

		if err != nil {
			// fmt.Printf("debug readbytes: %T %v\n", err, err)
			o.conn.Close()
			return nil
		}

		if len(record) == 0 {
			// fmt.Printf("debug len(line) == 0\n")
			return nil
		}

		switch record[0] {
		case "CLIENT_LIST":
			if len(record) != len(clientListFields) {
				return fmt.Errorf("expected 12 columns")
			}

			m, err := metric.New("openvpn", nil, nil, time.Now())
			if err != nil {
				return err
			}

			for i, col := range clientListFields {
				col.valueFunc(col.key, record[i], m)
			}

			acc.AddMetric(m)
		case "END":
			return nil
		}
	}
}

func (o *OpenVPN) Stop() error {
	if o.conn != nil {
		_, err := o.conn.Write([]byte("quit\n"))
		o.conn.Close()
		return err
	}
	return nil
}

func init() {
	inputs.Add("openvpn", func() telegraf.Input {
		return &OpenVPN{
			Timeout: config.Duration(defaultTimeout),
		}
	})
}

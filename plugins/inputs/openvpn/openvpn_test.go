package openvpn_test

import "testing"

var example = `>INFO:OpenVPN Management Interface Version 1 -- type 'help' for more info
TITLE,OpenVPN 2.4.6 x86_64-pc-linux-gnu [SSL (OpenSSL)] [LZO] [LZ4] [EPOLL] [MH/PKTINFO] [AEAD] built on Jul 14 2018
TIME,Sun Jan  6 03:29:09 2019,1546774149
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Virtual IPv6 Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username,Client ID,Peer ID
CLIENT_LIST,loaner.lan,10.13.49.1:53864,,,14520,22066,Sun Jan  6 03:04:18 2019,1546772658,UNDEF,72,0
HEADER,ROUTING_TABLE,Virtual Address,Common Name,Real Address,Last Ref,Last Ref (time_t)
ROUTING_TABLE,52:54:00:5b:93:55,loaner.lan,10.13.49.1:53864,Sun Jan  6 03:28:40 2019,1546774120
GLOBAL_STATS,Max bcast/mcast queue length,5
END
`

func Test(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

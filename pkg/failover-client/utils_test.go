package failoverclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseValueFromInfo(t *testing.T) {
	info := "txt:# Replication\r\nrole:master\r\nconnected_slaves:2\r\nslave0:ip=10.88.0.170,port=6379,state=wait_bgsave,offset=0,lag=0,type=replica\r\nslave1:ip=10.88.0.171,port=6379,state=wait_bgsave,offset=0,lag=0,type=replica\r\nreplicas_waiting_psync:0\r\nmaster_failover_state:no-failover\r\nmaster_replid:240bcba5fe13f68d5fa1d9ab84e3e3878b68552a\r\nmaster_replid2:0000000000000000000000000000000000000000\r\nmaster_repl_offset:0\r\nsecond_repl_offset:-1\r\nrepl_backlog_active:1\r\nrepl_backlog_size:10485760\r\nrepl_backlog_first_byte_offset:1\r\nrepl_backlog_histlen:0\r\n"
	assert := assert.New(t)

	assert.Equal(master, ParseValueFromInfo(info, role))

	assert.Equal(master, ParseValueFromInfo("\r\ntest\r\nrole:master\r\nconnected_slaves:2", role), "Should not panic when split does not work correctly")

	assert.Equal("", ParseValueFromInfo("", "not-a-key"), "Should return an empty string if no value is found")
}

func TestExtractPortFromAddress(t *testing.T) {
	tMatrix := map[string]struct {
		address     string
		defaultPort int64
		expHost     string
		expPort     int64
	}{
		"WithPort": {
			address:     "node1:6379",
			defaultPort: 1234,
			expHost:     "node1",
			expPort:     6379,
		},
		"NoPort": {
			address:     "node2",
			defaultPort: 1234,
			expHost:     "node2",
			expPort:     1234,
		},
		"IPv6WithPort": {
			address:     "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:6380",
			defaultPort: 1234,
			expHost:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expPort:     6380,
		},
		"IPv6NoPort": {
			address:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			defaultPort: 1234,
			expHost:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expPort:     1234,
		},
	}

	for name, tCase := range tMatrix {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			host, port := extractPortFromAddress(tCase.address, tCase.defaultPort)
			assert.Equal(tCase.expHost, host, "Should extract expected host")
			assert.Equal(tCase.expPort, port, "Should extract expected port")
		})
	}
}

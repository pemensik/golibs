package netutil_test

import (
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneIPs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []net.IP(nil), netutil.CloneIPs(nil))
	assert.Equal(t, []net.IP{}, netutil.CloneIPs([]net.IP{}))

	ips := []net.IP{testIPv4}
	clone := netutil.CloneIPs(ips)
	assert.Equal(t, ips, clone)

	require.Len(t, clone, len(ips))
	require.Len(t, clone[0], len(ips[0]))

	assert.NotSame(t, &ips[0], &clone[0])
	assert.NotSame(t, &ips[0][0], &clone[0][0])
}

func TestSpecialAddrs(t *testing.T) {
	t.Parallel()

	assert.NotSame(t, netutil.IPv4bcast(), netutil.IPv4bcast())
	assert.NotSame(t, netutil.IPv4allsys(), netutil.IPv4allsys())
	assert.NotSame(t, netutil.IPv4allrouter(), netutil.IPv4allrouter())

	assert.NotSame(t, netutil.IPv4Zero(), netutil.IPv4Zero())
	assert.NotSame(t, netutil.IPv6Zero(), netutil.IPv6Zero())
}

func TestIPAndPortFromAddr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		in       net.Addr
		wantIP   net.IP
		wantPort uint16
	}{{
		name:     "nil",
		in:       nil,
		wantIP:   nil,
		wantPort: 0,
	}, {
		name:     "tcp",
		in:       &net.TCPAddr{IP: testIPv4, Port: 12345},
		wantIP:   testIPv4,
		wantPort: 12345,
	}, {
		name:     "udp",
		in:       &net.UDPAddr{IP: testIPv4, Port: 12345},
		wantIP:   testIPv4,
		wantPort: 12345,
	}, {
		name:     "custom",
		in:       struct{ net.Addr }{},
		wantIP:   nil,
		wantPort: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotIP, gotPort := netutil.IPAndPortFromAddr(tc.in)
			assert.Equal(t, tc.wantIP, gotIP)
			assert.Equal(t, tc.wantPort, gotPort)
		})
	}
}

func TestValidateIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		wantErrMsg string
		wantErrAs  any
		in         net.IP
	}{{
		name:       "success_ipv4",
		wantErrMsg: "",
		wantErrAs:  nil,
		in:         testIPv4,
	}, {
		name:       "success_ipv6",
		wantErrMsg: "",
		wantErrAs:  nil,
		in:         testIPv6,
	}, {
		name:       "error_nil",
		wantErrMsg: `bad ip address "<nil>": ip address is empty`,
		wantErrAs:  new(*netutil.LengthError),
		in:         nil,
	}, {
		name:       "error_empty",
		wantErrMsg: `bad ip address "<nil>": ip address is empty`,
		wantErrAs:  new(*netutil.LengthError),
		in:         net.IP{},
	}, {
		name: "error_bad",
		wantErrMsg: `bad ip address "?010203": ` +
			`bad ip address length 3, allowed: [4 16]`,
		wantErrAs: new(*netutil.LengthError),
		in:        net.IP{1, 2, 3},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := netutil.ValidateIP(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			if tc.wantErrAs != nil {
				require.Error(t, err)

				assert.ErrorAs(t, err, new(*netutil.AddrError))
				assert.ErrorAs(t, err, tc.wantErrAs)
			}
		})
	}
}

func TestIsValidIPString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want assert.BoolAssertionFunc
		name string
		in   string
	}{{
		want: assert.True,
		name: "good_ipv4",
		in:   testIPv4.String(),
	}, {
		want: assert.True,
		name: "good_ipv6",
		in:   testIPv6.String(),
	}, {
		want: assert.True,
		name: "good_ipv6_unspec",
		in:   "::",
	}, {
		want: assert.True,
		name: "good_4in6",
		in:   "::ffff:192.168.140.255",
	}, {
		want: assert.True,
		name: "good_ipv6_zone",
		in:   "fd7a:115c:a1e0:ab12:4843:cd96:626b:430b%eth0",
	}, {
		want: assert.True,
		name: "good_ipv6_ellipsis",
		in:   "fd7a:115c:a1e0:ab12:4843:cd96:626b::",
	}, {
		want: assert.True,
		name: "good_ipv6_leading_zeros",
		in:   "000000::",
	}, {
		want: assert.False,
		name: "bad_colon",
		in:   ":",
	}, {
		want: assert.False,
		name: "not_ip",
		in:   "not_ip",
	}, {
		want: assert.False,
		name: "bad_ipv4_short",
		in:   "1.2.3",
	}, {
		want: assert.False,
		name: "bad_ipv4_chars",
		in:   "1.ff.3.4",
	}, {
		want: assert.False,
		name: "bad_ipv4_long",
		in:   "1.2.3.4.5",
	}, {
		want: assert.False,
		name: "bad_ipv4_label",
		in:   "1.2.3.4567",
	}, {
		want: assert.False,
		name: "bad_ipv4_leading_zero",
		in:   "1.2.3.04",
	}, {
		want: assert.False,
		name: "bad_ipv6_separator",
		in:   "1::2.3",
	}, {
		want: assert.False,
		name: "bad_ipv6_ellipsis",
		in:   "fd7a:115c:a1e0:ab12:4843:cd96:626b::430b",
	}, {
		want: assert.False,
		name: "bad_ipv6_many_ellipses",
		in:   "::cd96::626b::",
	}, {
		want: assert.False,
		name: "bad_ipv6_overflow",
		in:   "::fffff",
	}, {
		want: assert.False,
		name: "bad_ipv6_separator_position",
		in:   "::626b:430b:",
	}, {
		want: assert.False,
		name: "bad_ipv6_empty_zone",
		in:   "::%",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.want(t, netutil.IsValidIPString(tc.in))
		})
	}
}

func BenchmarkIsValidIPString(b *testing.B) {
	benchCases := []struct {
		want require.BoolAssertionFunc
		name string
		in   string
	}{{
		want: require.True,
		name: "good_ipv4",
		in:   "0.0.0.0",
	}, {
		want: require.True,
		name: "good_ipv4_long",
		in:   "255.255.255.255",
	}, {
		want: require.True,
		name: "good_ipv6",
		in:   "2001:db8::",
	}, {
		want: require.True,
		name: "good_ipv6_long",
		in:   "2001:db8:a1e0:ab12:4843:cd96:626b::",
	}, {
		want: require.False,
		name: "not_ip",
		in:   strings.Repeat("a", 256),
	}, {
		want: require.False,
		name: "zeroes",
		in:   strings.Repeat("0", 256),
	}, {
		want: require.False,
		name: "bad_ipv4",
		in:   "1.2.3",
	}, {
		want: require.False,
		name: "bad_ipv4_long",
		in:   "255.255.255.256",
	}, {
		want: require.False,
		name: "bad_ipv6",
		in:   "2001:db8:::",
	}, {
		want: require.False,
		name: "bad_ipv6_long",
		in:   "2001:db8:a1e0:ab12:4843:cd96:626b:ffff:ffff",
	}}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				boolSink = netutil.IsValidIPString(bc.in)
			}

			bc.want(b, boolSink)
		})
	}

	// goos: darwin
	// goarch: amd64
	// pkg: github.com/AdguardTeam/golibs/netutil
	// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
	// BenchmarkIsValidIPString/good_ipv4-12		35469972	33.93 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/good_ipv4_long-12	23231380	51.57 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/good_ipv6-12		24500007	46.62 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/good_ipv6_long-12	9725222		126.7 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/not_ip-12			170261018	7.078 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/zeroes-12			6189428		189.4 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/bad_ipv4-12			44662938	26.39 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/bad_ipv4_long-12	21964128	49.89 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/bad_ipv6-12			22520074	53.08 ns/op		0 B/op	0 allocs/op
	// BenchmarkIsValidIPString/bad_ipv6_long-12	8275022		147.3 ns/op		0 B/op	0 allocs/op
}

func FuzzIsValidIPString(f *testing.F) {
	for _, seed := range []string{
		"",
		" ",
		"192.0.2.1",
		"2001:db8::68",
		"::ffff:192.168.140.255",
		"1.2.3",
		"1::2.3",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		ok := netutil.IsValidIPString(input)
		_, err := netip.ParseAddr(input)

		require.Equal(t, err == nil, ok)
	})
}

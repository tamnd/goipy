import ipaddress


def test_ipv4_address_from_string():
    a = ipaddress.IPv4Address('192.168.1.1')
    print(repr(a))
    print(str(a))
    print(a.packed)
    print(a.version)
    print(a.max_prefixlen)
    print(a.compressed)
    print('test_ipv4_address_from_string ok')


def test_ipv4_address_from_int():
    a = ipaddress.IPv4Address(0xC0A80101)  # 192.168.1.1
    print(str(a))
    b = ipaddress.IPv4Address(2130706433)  # 127.0.0.1
    print(str(b))
    print('test_ipv4_address_from_int ok')


def test_ipv4_address_properties():
    private = ipaddress.IPv4Address('192.168.1.1')
    print(private.is_private)
    print(private.is_global)

    loopback = ipaddress.IPv4Address('127.0.0.1')
    print(loopback.is_loopback)
    print(loopback.is_private)

    multicast = ipaddress.IPv4Address('224.0.0.1')
    print(multicast.is_multicast)

    link_local = ipaddress.IPv4Address('169.254.1.1')
    print(link_local.is_link_local)

    unspecified = ipaddress.IPv4Address('0.0.0.0')
    print(unspecified.is_unspecified)

    public = ipaddress.IPv4Address('8.8.8.8')
    print(public.is_global)
    print('test_ipv4_address_properties ok')


def test_ipv4_address_arith():
    a = ipaddress.IPv4Address('10.0.0.1')
    b = a + 1
    print(str(b))
    c = a + 255
    print(str(c))
    d = ipaddress.IPv4Address('10.0.1.0')
    e = d - 1
    print(str(e))
    print(a == ipaddress.IPv4Address('10.0.0.1'))
    print(a < d)
    print(d > a)
    print(int(a))
    print('test_ipv4_address_arith ok')


def test_ipv4_network_basic():
    n = ipaddress.IPv4Network('192.168.1.0/24')
    print(repr(n))
    print(str(n))
    print(str(n.network_address))
    print(str(n.broadcast_address))
    print(str(n.netmask))
    print(str(n.hostmask))
    print(n.prefixlen)
    print(n.num_addresses)
    print(n.version)
    print(n.max_prefixlen)
    print(n.with_prefixlen)
    print(n.with_netmask)
    print(n.with_hostmask)
    print(n.compressed)
    print('test_ipv4_network_basic ok')


def test_ipv4_network_contains():
    n = ipaddress.IPv4Network('192.168.1.0/24')
    print(ipaddress.IPv4Address('192.168.1.1') in n)
    print(ipaddress.IPv4Address('192.168.1.255') in n)
    print(ipaddress.IPv4Address('192.168.2.1') in n)
    print(ipaddress.IPv4Address('192.168.0.255') in n)
    print('test_ipv4_network_contains ok')


def test_ipv4_network_hosts():
    n = ipaddress.IPv4Network('192.168.1.0/24')
    hosts = list(n.hosts())
    print(len(hosts))
    print(str(hosts[0]))
    print(str(hosts[-1]))

    n2 = ipaddress.IPv4Network('10.0.0.0/30')
    hosts2 = list(n2.hosts())
    print(len(hosts2))
    print(str(hosts2[0]))
    print(str(hosts2[-1]))
    print('test_ipv4_network_hosts ok')


def test_ipv4_network_iter():
    n = ipaddress.IPv4Network('10.0.0.0/30')
    addrs = list(n)
    print(len(addrs))
    print(str(addrs[0]))
    print(str(addrs[-1]))
    print('test_ipv4_network_iter ok')


def test_ipv4_network_subnets():
    n = ipaddress.IPv4Network('192.168.1.0/24')
    subs = list(n.subnets())
    print(len(subs))
    print(str(subs[0]))
    print(str(subs[1]))

    subs2 = list(n.subnets(prefixlen_diff=2))
    print(len(subs2))
    print(str(subs2[0]))

    subs3 = list(n.subnets(new_prefix=26))
    print(len(subs3))
    print('test_ipv4_network_subnets ok')


def test_ipv4_network_supernet():
    n = ipaddress.IPv4Network('192.168.1.0/24')
    sup = n.supernet()
    print(str(sup))
    sup2 = n.supernet(prefixlen_diff=2)
    print(str(sup2))
    print('test_ipv4_network_supernet ok')


def test_ipv4_network_subnet_of():
    small = ipaddress.IPv4Network('192.168.1.0/25')
    big = ipaddress.IPv4Network('192.168.1.0/24')
    print(small.subnet_of(big))
    print(big.subnet_of(small))
    print(big.supernet_of(small))
    print(small.supernet_of(big))
    print('test_ipv4_network_subnet_of ok')


def test_ipv4_network_overlaps():
    a = ipaddress.IPv4Network('192.168.1.0/24')
    b = ipaddress.IPv4Network('192.168.1.128/25')
    c = ipaddress.IPv4Network('192.168.2.0/24')
    print(a.overlaps(b))
    print(a.overlaps(c))
    print(b.overlaps(a))
    print('test_ipv4_network_overlaps ok')


def test_ipv4_network_strict():
    try:
        ipaddress.IPv4Network('192.168.1.5/24', strict=True)
        print('no error')
    except ValueError:
        print('ValueError raised')

    n = ipaddress.IPv4Network('192.168.1.5/24', strict=False)
    print(str(n))
    print('test_ipv4_network_strict ok')


def test_ipv4_interface():
    iface = ipaddress.IPv4Interface('192.168.1.5/24')
    print(repr(iface))
    print(str(iface))
    print(str(iface.ip))
    print(str(iface.network))
    print(str(iface.netmask))
    print(iface.with_prefixlen)
    print(iface.with_netmask)
    print(iface.with_hostmask)
    print('test_ipv4_interface ok')


def test_ipv6_address():
    a = ipaddress.IPv6Address('2001:db8::1')
    print(repr(a))
    print(str(a))
    print(a.version)
    print(a.max_prefixlen)
    print(a.compressed)
    print(a.exploded)
    print('test_ipv6_address ok')


def test_ipv6_address_properties():
    loopback = ipaddress.IPv6Address('::1')
    print(loopback.is_loopback)
    print(loopback.is_private)

    link_local = ipaddress.IPv6Address('fe80::1')
    print(link_local.is_link_local)
    print(link_local.is_private)

    multicast = ipaddress.IPv6Address('ff02::1')
    print(multicast.is_multicast)

    unspecified = ipaddress.IPv6Address('::')
    print(unspecified.is_unspecified)

    public = ipaddress.IPv6Address('2001:db8::1')
    print(public.is_global)
    print('test_ipv6_address_properties ok')


def test_ipv6_address_ipv4_mapped():
    mapped = ipaddress.IPv6Address('::ffff:192.168.1.1')
    print(mapped.ipv4_mapped)
    print(str(mapped.ipv4_mapped))

    not_mapped = ipaddress.IPv6Address('2001:db8::1')
    print(not_mapped.ipv4_mapped)
    print('test_ipv6_address_ipv4_mapped ok')


def test_ipv6_network():
    n = ipaddress.IPv6Network('2001:db8::/32')
    print(repr(n))
    print(str(n))
    print(n.prefixlen)
    print(n.version)
    print(str(n.network_address))
    print(str(n.broadcast_address))
    print(n.compressed)
    print(n.with_prefixlen)

    subs = list(n.subnets())
    print(len(subs))
    print(str(subs[0]))

    sup = n.supernet()
    print(str(sup))
    print('test_ipv6_network ok')


def test_ipv6_interface():
    iface = ipaddress.IPv6Interface('2001:db8::1/32')
    print(repr(iface))
    print(str(iface))
    print(str(iface.ip))
    print(str(iface.network))
    print(iface.with_prefixlen)
    print('test_ipv6_interface ok')


def test_factory_functions():
    a4 = ipaddress.ip_address('192.168.1.1')
    print(type(a4).__name__)
    print(str(a4))

    a6 = ipaddress.ip_address('::1')
    print(type(a6).__name__)
    print(str(a6))

    n4 = ipaddress.ip_network('10.0.0.0/8')
    print(type(n4).__name__)
    print(str(n4))

    n6 = ipaddress.ip_network('2001:db8::/32')
    print(type(n6).__name__)
    print(str(n6))

    i4 = ipaddress.ip_interface('192.168.1.5/24')
    print(type(i4).__name__)
    print(str(i4))

    i6 = ipaddress.ip_interface('2001:db8::1/32')
    print(type(i6).__name__)
    print(str(i6))
    print('test_factory_functions ok')


def test_errors():
    try:
        ipaddress.IPv4Address('not_an_ip')
    except ipaddress.AddressValueError:
        print('AddressValueError raised')
    except ValueError:
        print('ValueError raised')

    try:
        ipaddress.IPv4Network('192.168.1.0/33')
    except ipaddress.NetmaskValueError:
        print('NetmaskValueError raised')
    except ValueError:
        print('ValueError raised')

    try:
        ipaddress.IPv6Address('not::valid::addr::here::x')
    except ipaddress.AddressValueError:
        print('AddressValueError raised for ipv6')
    except ValueError:
        print('ValueError raised for ipv6')

    print('test_errors ok')


def test_constants():
    print(ipaddress.IPV4LENGTH)
    print(ipaddress.IPV6LENGTH)
    print('test_constants ok')


def test_v4_int_to_packed():
    b = ipaddress.v4_int_to_packed(0xC0A80101)  # 192.168.1.1
    print(b)
    print(len(b))
    b2 = ipaddress.v4_int_to_packed(0)
    print(b2)
    print('test_v4_int_to_packed ok')


def test_get_mixed_type_key():
    a4 = ipaddress.IPv4Address('192.168.1.1')
    key = ipaddress.get_mixed_type_key(a4)
    print(type(key).__name__)
    print(len(key))
    print(key[0])

    a6 = ipaddress.IPv6Address('::1')
    key6 = ipaddress.get_mixed_type_key(a6)
    print(key6[0])
    print('test_get_mixed_type_key ok')


def test_summarize_address_range():
    first = ipaddress.IPv4Address('192.168.1.0')
    last = ipaddress.IPv4Address('192.168.1.255')
    networks = list(ipaddress.summarize_address_range(first, last))
    print(len(networks))
    print(str(networks[0]))
    print('test_summarize_address_range ok')


test_ipv4_address_from_string()
test_ipv4_address_from_int()
test_ipv4_address_properties()
test_ipv4_address_arith()
test_ipv4_network_basic()
test_ipv4_network_contains()
test_ipv4_network_hosts()
test_ipv4_network_iter()
test_ipv4_network_subnets()
test_ipv4_network_supernet()
test_ipv4_network_subnet_of()
test_ipv4_network_overlaps()
test_ipv4_network_strict()
test_ipv4_interface()
test_ipv6_address()
test_ipv6_address_properties()
test_ipv6_address_ipv4_mapped()
test_ipv6_network()
test_ipv6_interface()
test_factory_functions()
test_errors()
test_constants()
test_v4_int_to_packed()
test_get_mixed_type_key()
test_summarize_address_range()

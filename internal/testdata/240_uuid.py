import uuid

# 1 constants
print(uuid.RESERVED_NCS)
print(uuid.RFC_4122)
print(uuid.RESERVED_MICROSOFT)
print(uuid.RESERVED_FUTURE)

# 2 safe_uuid_values
print(uuid.SafeUUID.safe.value)
print(uuid.SafeUUID.unsafe.value)
print(uuid.SafeUUID.unknown.value)

# 3 safe_uuid_str
print(str(uuid.SafeUUID.safe))
print(str(uuid.SafeUUID.unsafe))
print(str(uuid.SafeUUID.unknown))

# 4 safe_uuid_isinstance
print(isinstance(uuid.SafeUUID.safe, uuid.SafeUUID))
print(isinstance(uuid.SafeUUID.unsafe, uuid.SafeUUID))

# 5 uuid_from_string
u = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')
print(u.hex)
print(u.int)
print(u.variant)
print(u.version)
print(u.urn)
print(u.time_low)
print(u.time_mid)
print(u.time_hi_version)
print(u.clock_seq_hi_variant)
print(u.clock_seq_low)
print(u.node)
print(u.clock_seq)

# 6 uuid_from_hex_kwarg
u2 = uuid.UUID(hex='6ba7b810-9dad-11d1-80b4-00c04fd430c8')
print(u2.hex)

# 7 uuid_from_int_kwarg
u3 = uuid.UUID(int=143098242404177361603877621312831893704)
print(u3.hex)

# 8 uuid_from_bytes_kwarg
u4 = uuid.UUID(bytes=u.bytes)
print(u4.hex)

# 9 uuid_from_fields_kwarg
u5 = uuid.UUID(fields=u.fields)
print(u5.hex)

# 10 uuid_from_bytes_le_kwarg
u6 = uuid.UUID(bytes_le=u.bytes_le)
print(u6.hex)

# 11 uuid_equality
ua = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')
ub = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')
uc = uuid.UUID('6ba7b811-9dad-11d1-80b4-00c04fd430c8')
print(ua == ub)
print(ua == uc)
print(ua != uc)

# 12 uuid_comparison
print(ua < uc)
print(ua > uc)
print(ua <= ub)
print(ua >= ub)

# 13 uuid_hash
uh1 = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')
uh2 = uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')
uh3 = uuid.UUID('6ba7b810-9dad-11d1-80b5-00c04fd430c8')
print(hash(uh1) == hash(uh2))
print(hash(uh1) == hash(uh3))

# 14 uuid_str_and_repr
print(str(ua))
print(repr(ua))

# 15 uuid4
u4r = uuid.uuid4()
print(u4r.version)
print(u4r.variant)
print(isinstance(u4r, uuid.UUID))

# 16 uuid3 deterministic
u3d = uuid.uuid3(uuid.NAMESPACE_DNS, 'python.org')
print(u3d)
print(u3d.version)
print(u3d.variant)

# 17 uuid5 deterministic
u5d = uuid.uuid5(uuid.NAMESPACE_DNS, 'python.org')
print(u5d)
print(u5d.version)
print(u5d.variant)

# 18 uuid1
u1 = uuid.uuid1()
print(u1.version)
print(u1.variant)
print(isinstance(u1, uuid.UUID))

# 19 namespace_constants
print(uuid.NAMESPACE_DNS)
print(uuid.NAMESPACE_URL)
print(uuid.NAMESPACE_OID)
print(uuid.NAMESPACE_X500)

# 20 getnode
node = uuid.getnode()
print(type(node).__name__)
print(node > 0)

# 21 module_exports
for name in ['UUID', 'SafeUUID', 'NAMESPACE_DNS', 'NAMESPACE_URL', 'NAMESPACE_OID',
             'NAMESPACE_X500', 'RESERVED_NCS', 'RFC_4122', 'RESERVED_MICROSOFT',
             'RESERVED_FUTURE', 'uuid1', 'uuid3', 'uuid4', 'uuid5', 'getnode']:
    print(name, name in dir(uuid))

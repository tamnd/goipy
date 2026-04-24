import hmac
import hashlib

# new() + hexdigest() for sha256
h = hmac.new(b"key", b"hello", digestmod="sha256")
print("sha256:", h.hexdigest())

# update chaining equals single call
h = hmac.new(b"key", digestmod="sha256")
h.update(b"hel")
h.update(b"lo")
print("update chain:", h.hexdigest())

# digest() method returns bytes
h = hmac.new(b"key", b"hello", digestmod="sha256")
print("digest bytes:", h.digest().hex())

# name attribute
h = hmac.new(b"key", b"hello", digestmod="sha256")
print("name:", h.name)

# digest_size attribute
h = hmac.new(b"key", b"hello", digestmod="sha256")
print("digest_size:", h.digest_size)

# block_size attribute
h = hmac.new(b"key", b"hello", digestmod="sha256")
print("block_size:", h.block_size)

# copy() - modify copy doesn't affect original
h = hmac.new(b"key", b"hello", digestmod="sha256")
h2 = h.copy()
h2.update(b" world")
print("orig after copy:", h.hexdigest())
print("copy updated:", h2.hexdigest())

# digestmod as callable (hashlib constructor)
h = hmac.new(b"key", b"hello", digestmod=hashlib.sha256)
print("callable digestmod:", h.hexdigest())

# hmac.digest() one-shot
d = hmac.digest(b"key", b"hello", "sha256")
print("digest one-shot:", d.hex())

# compare_digest bytes
print("compare eq bytes:", hmac.compare_digest(b"abc", b"abc"))
print("compare ne bytes:", hmac.compare_digest(b"abc", b"xyz"))

# compare_digest str
print("compare eq str:", hmac.compare_digest("abc", "abc"))
print("compare ne str:", hmac.compare_digest("abc", "xyz"))

# sha1
h = hmac.new(b"key", b"hello", digestmod="sha1")
print("sha1:", h.hexdigest())
print("sha1 block_size:", h.block_size)
print("sha1 digest_size:", h.digest_size)

# sha512
h = hmac.new(b"key", b"hello", digestmod="sha512")
print("sha512:", h.hexdigest())
print("sha512 block_size:", h.block_size)
print("sha512 digest_size:", h.digest_size)

# md5
h = hmac.new(b"key", b"hello", digestmod="md5")
print("md5:", h.hexdigest())
print("md5 block_size:", h.block_size)
print("md5 digest_size:", h.digest_size)

# sha3_256
h = hmac.new(b"key", b"hello", digestmod="sha3_256")
print("sha3_256:", h.hexdigest())
print("sha3_256 block_size:", h.block_size)
print("sha3_256 digest_size:", h.digest_size)

# sha3_512
h = hmac.new(b"key", b"hello", digestmod="sha3_512")
print("sha3_512:", h.hexdigest())
print("sha3_512 block_size:", h.block_size)
print("sha3_512 digest_size:", h.digest_size)

# long key
h = hmac.new(b"a" * 200, b"hello", digestmod="sha256")
print("long key:", h.hexdigest())

# empty message
h = hmac.new(b"key", b"", digestmod="sha256")
print("empty msg:", h.hexdigest())

# digest one-shot with sha1
d = hmac.digest(b"key", b"hello", "sha1")
print("sha1 one-shot:", d.hex())

# digestmod as hashlib.sha1 callable
h = hmac.new(b"key", b"hello", digestmod=hashlib.sha1)
print("sha1 callable:", h.hexdigest())

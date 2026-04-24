import hashlib

# basic digest/hexdigest for all algorithms
print("=== basic ===")
for algo in ["md5", "sha1", "sha224", "sha256", "sha384", "sha512"]:
    h = hashlib.new(algo, b"hello")
    print(f"{algo}: {h.hexdigest()}")

print()
print("=== sha3 ===")
for algo in ["sha3_224", "sha3_256", "sha3_384", "sha3_512"]:
    h = hashlib.new(algo, b"hello")
    print(f"{algo}: {h.hexdigest()}")

print()
print("=== shake ===")
h = hashlib.shake_128(b"hello")
print(f"shake_128 32: {h.hexdigest(32)}")
h = hashlib.shake_256(b"hello")
print(f"shake_256 32: {h.hexdigest(32)}")

print()
print("=== blake2 ===")
h = hashlib.blake2b(b"hello")
print(f"blake2b: {h.hexdigest()}")
h = hashlib.blake2s(b"hello")
print(f"blake2s: {h.hexdigest()}")
h = hashlib.blake2b(b"hello", digest_size=32)
print(f"blake2b-32: {h.hexdigest()}")
h = hashlib.blake2s(b"hello", digest_size=16, key=b"secretkey12345ab")
print(f"blake2s-16: {h.hexdigest()}")

print()
print("=== update chaining ===")
h = hashlib.sha256()
h.update(b"he")
h.update(b"llo")
print(f"sha256 update: {h.hexdigest()}")

print()
print("=== copy ===")
h = hashlib.sha256(b"hello")
h2 = h.copy()
h2.update(b" world")
print(f"original: {h.hexdigest()}")
print(f"copied: {h2.hexdigest()}")

print()
print("=== digest bytes ===")
h = hashlib.md5(b"hello")
print(f"md5 digest: {h.digest().hex()}")

print()
print("=== attributes ===")
h = hashlib.sha256()
print(f"name: {h.name}")
print(f"digest_size: {h.digest_size}")
print(f"block_size: {h.block_size}")
h = hashlib.md5()
print(f"md5 block_size: {h.block_size}")
h = hashlib.sha1()
print(f"sha1 block_size: {h.block_size}")
h = hashlib.sha512()
print(f"sha512 block_size: {h.block_size}")
h = hashlib.sha3_256()
print(f"sha3_256 block_size: {h.block_size}")

print()
print("=== algorithms_guaranteed ===")
print(f"in guaranteed: {'sha256' in hashlib.algorithms_guaranteed}")
print(f"in guaranteed: {'md5' in hashlib.algorithms_guaranteed}")
print(f"in guaranteed: {'sha3_256' in hashlib.algorithms_guaranteed}")

print()
print("=== algorithms_available ===")
print(f"in available: {'sha256' in hashlib.algorithms_available}")

print()
print("=== usedforsecurity kwarg ===")
h = hashlib.md5(b"hello", usedforsecurity=False)
print(f"usedforsecurity: {h.hexdigest()}")

print()
print("=== new() dispatch ===")
h = hashlib.new("sha256", b"hello")
print(f"new sha256: {h.hexdigest()}")

print()
print("=== file_digest ===")
import io
data = b"hello world"
f = io.BytesIO(data)
h = hashlib.file_digest(f, "sha256")
print(f"file_digest: {h.hexdigest()}")

print()
print("=== compare_digest ===")
import hmac
print(f"compare equal: {hmac.compare_digest(b'abc', b'abc')}")
print(f"compare unequal: {hmac.compare_digest(b'abc', b'def')}")

print()
print("=== blake2 with key ===")
h = hashlib.blake2b(b"hello", key=b"secret")
print(f"blake2b keyed: {h.hexdigest()}")

print()
print("=== shake copy ===")
h = hashlib.shake_256(b"hello")
h2 = h.copy()
h2.update(b" world")
print(f"shake orig: {h.hexdigest(16)}")
print(f"shake copy: {h2.hexdigest(16)}")

print()
print("=== sha3 digest bytes ===")
h = hashlib.sha3_256(b"hello")
print(f"sha3_256 digest: {h.digest().hex()}")

print()
print("=== direct constructors ===")
print(hashlib.md5(b"hello").hexdigest())
print(hashlib.sha1(b"hello").hexdigest())
print(hashlib.sha256(b"hello").hexdigest())
print(hashlib.sha3_256(b"hello").hexdigest())
print(hashlib.shake_128(b"hello").hexdigest(8))

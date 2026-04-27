import ensurepip


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# ── import ────────────────────────────────────────────────────────────────────

def test_import():
    import ensurepip as e
    assert e is not None

run("test_import", test_import)


# ── version() ─────────────────────────────────────────────────────────────────

def test_version_returns_str():
    v = ensurepip.version()
    assert isinstance(v, str), f"expected str, got {type(v)}"

run("test_version_returns_str", test_version_returns_str)


def test_version_not_empty():
    v = ensurepip.version()
    assert len(v) > 0

run("test_version_not_empty", test_version_not_empty)


def test_version_has_dot():
    v = ensurepip.version()
    assert "." in v, f"expected version with dot, got {v!r}"

run("test_version_has_dot", test_version_has_dot)


# ── bootstrap() ───────────────────────────────────────────────────────────────

def test_bootstrap_no_args():
    ensurepip.bootstrap()  # no-op stub, no crash

run("test_bootstrap_no_args", test_bootstrap_no_args)


def test_bootstrap_root():
    ensurepip.bootstrap(root="/tmp")

run("test_bootstrap_root", test_bootstrap_root)


def test_bootstrap_upgrade():
    ensurepip.bootstrap(upgrade=True)

run("test_bootstrap_upgrade", test_bootstrap_upgrade)


def test_bootstrap_user():
    ensurepip.bootstrap(user=True)

run("test_bootstrap_user", test_bootstrap_user)


def test_bootstrap_default_pip():
    ensurepip.bootstrap(default_pip=True)

run("test_bootstrap_default_pip", test_bootstrap_default_pip)


def test_bootstrap_altinstall():
    ensurepip.bootstrap(altinstall=True)

run("test_bootstrap_altinstall", test_bootstrap_altinstall)


def test_bootstrap_verbosity():
    ensurepip.bootstrap(verbosity=2)

run("test_bootstrap_verbosity", test_bootstrap_verbosity)


def test_bootstrap_returns_none():
    result = ensurepip.bootstrap()
    assert result is None

run("test_bootstrap_returns_none", test_bootstrap_returns_none)


def test_bootstrap_altinstall_and_default_pip_raises():
    try:
        ensurepip.bootstrap(altinstall=True, default_pip=True)
        assert False, "expected ValueError"
    except ValueError:
        pass

run("test_bootstrap_altinstall_and_default_pip_raises", test_bootstrap_altinstall_and_default_pip_raises)


# ── _bundled_packages() ───────────────────────────────────────────────────────

def test_bundled_packages_returns_dict():
    result = ensurepip._bundled_packages()
    assert isinstance(result, dict)

run("test_bundled_packages_returns_dict", test_bundled_packages_returns_dict)


def test_bundled_packages_has_pip():
    result = ensurepip._bundled_packages()
    assert "pip" in result

run("test_bundled_packages_has_pip", test_bundled_packages_has_pip)


def test_bundled_packages_pip_version_str():
    result = ensurepip._bundled_packages()
    assert isinstance(result["pip"], str)

run("test_bundled_packages_pip_version_str", test_bundled_packages_pip_version_str)

import venv


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# ── import ────────────────────────────────────────────────────────────────────

def test_import():
    import venv as v
    assert v is not None

run("test_import", test_import)


# ── EnvBuilder constructor ────────────────────────────────────────────────────

def test_envbuilder_defaults():
    b = venv.EnvBuilder()
    assert b is not None

run("test_envbuilder_defaults", test_envbuilder_defaults)


def test_envbuilder_system_site_packages():
    b = venv.EnvBuilder(system_site_packages=True)
    assert b.system_site_packages == True

run("test_envbuilder_system_site_packages", test_envbuilder_system_site_packages)


def test_envbuilder_clear():
    b = venv.EnvBuilder(clear=True)
    assert b.clear == True

run("test_envbuilder_clear", test_envbuilder_clear)


def test_envbuilder_symlinks():
    b = venv.EnvBuilder(symlinks=True)
    assert b.symlinks == True

run("test_envbuilder_symlinks", test_envbuilder_symlinks)


def test_envbuilder_upgrade():
    b = venv.EnvBuilder(upgrade=True)
    assert b.upgrade == True

run("test_envbuilder_upgrade", test_envbuilder_upgrade)


def test_envbuilder_with_pip():
    b = venv.EnvBuilder(with_pip=True)
    assert b.with_pip == True

run("test_envbuilder_with_pip", test_envbuilder_with_pip)


def test_envbuilder_prompt():
    b = venv.EnvBuilder(prompt="myenv")
    assert b.prompt == "myenv"

run("test_envbuilder_prompt", test_envbuilder_prompt)


def test_envbuilder_upgrade_deps():
    b = venv.EnvBuilder(upgrade_deps=True)
    assert b.upgrade_deps == True

run("test_envbuilder_upgrade_deps", test_envbuilder_upgrade_deps)


# ── methods exist ─────────────────────────────────────────────────────────────

def test_envbuilder_has_create():
    b = venv.EnvBuilder()
    assert callable(b.create)

run("test_envbuilder_has_create", test_envbuilder_has_create)


def test_envbuilder_has_ensure_directories():
    b = venv.EnvBuilder()
    assert callable(b.ensure_directories)

run("test_envbuilder_has_ensure_directories", test_envbuilder_has_ensure_directories)


def test_envbuilder_has_create_configuration():
    b = venv.EnvBuilder()
    assert callable(b.create_configuration)

run("test_envbuilder_has_create_configuration", test_envbuilder_has_create_configuration)


def test_envbuilder_has_setup_python():
    b = venv.EnvBuilder()
    assert callable(b.setup_python)

run("test_envbuilder_has_setup_python", test_envbuilder_has_setup_python)


def test_envbuilder_has_setup_scripts():
    b = venv.EnvBuilder()
    assert callable(b.setup_scripts)

run("test_envbuilder_has_setup_scripts", test_envbuilder_has_setup_scripts)


def test_envbuilder_has_post_setup():
    b = venv.EnvBuilder()
    assert callable(b.post_setup)

run("test_envbuilder_has_post_setup", test_envbuilder_has_post_setup)


def test_envbuilder_has_upgrade_dependencies():
    b = venv.EnvBuilder()
    assert callable(b.upgrade_dependencies)

run("test_envbuilder_has_upgrade_dependencies", test_envbuilder_has_upgrade_dependencies)


def test_envbuilder_has_install_scripts():
    b = venv.EnvBuilder()
    assert callable(b.install_scripts)

run("test_envbuilder_has_install_scripts", test_envbuilder_has_install_scripts)


def test_envbuilder_has_create_git_ignore_file():
    b = venv.EnvBuilder()
    assert callable(b.create_git_ignore_file)

run("test_envbuilder_has_create_git_ignore_file", test_envbuilder_has_create_git_ignore_file)


# ── ensure_directories ────────────────────────────────────────────────────────

def test_ensure_directories_returns_context():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        ctx = b.ensure_directories(env_dir)
        assert ctx is not None

run("test_ensure_directories_returns_context", test_ensure_directories_returns_context)


def test_ensure_directories_has_env_dir():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        ctx = b.ensure_directories(env_dir)
        assert hasattr(ctx, "env_dir")

run("test_ensure_directories_has_env_dir", test_ensure_directories_has_env_dir)


def test_ensure_directories_has_bin_path():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        ctx = b.ensure_directories(env_dir)
        assert hasattr(ctx, "bin_path")

run("test_ensure_directories_has_bin_path", test_ensure_directories_has_bin_path)


def test_ensure_directories_has_executable():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        ctx = b.ensure_directories(env_dir)
        assert hasattr(ctx, "executable")

run("test_ensure_directories_has_executable", test_ensure_directories_has_executable)


# ── create() stub ─────────────────────────────────────────────────────────────

def test_create_stub():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        result = b.create(env_dir)
        assert result is None

run("test_create_stub", test_create_stub)


def test_setup_methods_no_crash():
    import tempfile, os
    b = venv.EnvBuilder()
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        ctx = b.ensure_directories(env_dir)
        b.create_configuration(ctx)
        b.setup_python(ctx)
        b.setup_scripts(ctx)
        b.post_setup(ctx)

run("test_setup_methods_no_crash", test_setup_methods_no_crash)


# ── module-level create() ─────────────────────────────────────────────────────

def test_module_create_exists():
    assert callable(venv.create)

run("test_module_create_exists", test_module_create_exists)


def test_module_create_stub():
    import tempfile, os
    with tempfile.TemporaryDirectory() as d:
        env_dir = os.path.join(d, "myenv")
        result = venv.create(env_dir)
        assert result is None

run("test_module_create_stub", test_module_create_stub)

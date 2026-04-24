"""Tests for the logging.config module."""
import logging
import logging.config
import os
import tempfile

# --- dictConfig: FileHandler + Formatter ---
tmpf1 = tempfile.mktemp(suffix='.log')
cfg1 = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'simple': {'format': '%(levelname)s:%(name)s:%(message)s'},
    },
    'handlers': {
        'fh1': {
            'class': 'logging.FileHandler',
            'filename': tmpf1,
            'mode': 'w',
            'formatter': 'simple',
            'level': 'DEBUG',
        },
    },
    'loggers': {
        'app1': {
            'level': 'DEBUG',
            'handlers': ['fh1'],
            'propagate': False,
        },
    },
}
logging.config.dictConfig(cfg1)
log1 = logging.getLogger('app1')
log1.info('hello')
log1.error('world')
for h in list(log1.handlers):
    h.close()
with open(tmpf1) as f:
    lines1 = [l.strip() for l in f.readlines()]
print(lines1[0])  # INFO:app1:hello
print(lines1[1])  # ERROR:app1:world
os.remove(tmpf1)

# --- dictConfig: NullHandler + level check ---
cfg2 = {
    'version': 1,
    'disable_existing_loggers': False,
    'handlers': {
        'null': {'class': 'logging.NullHandler'},
    },
    'loggers': {
        'app2': {
            'level': 'WARNING',
            'handlers': ['null'],
            'propagate': False,
        },
    },
}
logging.config.dictConfig(cfg2)
log2 = logging.getLogger('app2')
print(log2.level == logging.WARNING)  # True
print(len(log2.handlers) >= 1)        # True

# --- dictConfig: root logger ---
tmpf3 = tempfile.mktemp(suffix='.log')
cfg3 = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'fmt3': {'format': '%(levelname)s %(message)s'},
    },
    'handlers': {
        'fh3': {
            'class': 'logging.FileHandler',
            'filename': tmpf3,
            'mode': 'w',
            'formatter': 'fmt3',
            'level': 'ERROR',
        },
    },
    'root': {
        'level': 'ERROR',
        'handlers': ['fh3'],
    },
}
logging.config.dictConfig(cfg3)
root = logging.getLogger('')
root.error('root error')
root.debug('root debug')  # suppressed
for h in list(root.handlers):
    try:
        h.close()
    except Exception:
        pass
with open(tmpf3) as f:
    lines3 = [l.strip() for l in f.readlines()]
print(len(lines3) == 1)   # True
print(lines3[0])           # ERROR root error
os.remove(tmpf3)

# --- dictConfig: filters on handler ---
tmpf4 = tempfile.mktemp(suffix='.log')
cfg4 = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'fmt4': {'format': '%(message)s'},
    },
    'filters': {
        'subsonly': {'name': 'app4.sub'},
    },
    'handlers': {
        'fh4': {
            'class': 'logging.FileHandler',
            'filename': tmpf4,
            'mode': 'w',
            'formatter': 'fmt4',
            'filters': ['subsonly'],
        },
    },
    'loggers': {
        'app4': {
            'level': 'DEBUG',
            'handlers': ['fh4'],
            'propagate': False,
        },
    },
}
logging.config.dictConfig(cfg4)
app4 = logging.getLogger('app4')
app4sub = logging.getLogger('app4.sub')
app4sub.propagate = True
app4sub.setLevel(logging.DEBUG)
app4.info('blocked')       # filter blocks (name 'app4' != 'app4.sub')
app4sub.info('allowed')    # filter passes (name 'app4.sub')
for h in list(app4.handlers):
    h.close()
with open(tmpf4) as f:
    lines4 = [l.strip() for l in f.readlines()]
print(any('allowed' in l for l in lines4))         # True
print(not any('blocked' in l for l in lines4))     # True
os.remove(tmpf4)

# --- dictConfig: incremental ---
cfg5 = {'version': 1, 'incremental': True}
logging.config.dictConfig(cfg5)
print(True)  # no error

# --- fileConfig: NullHandler, root level set ---
ini1 = """\
[loggers]
keys=root

[handlers]
keys=nh

[formatters]
keys=

[logger_root]
level=INFO
handlers=nh

[handler_nh]
class=NullHandler
level=NOTSET
formatter=
args=()
"""
tmpini1 = tempfile.mktemp(suffix='.ini')
with open(tmpini1, 'w') as f:
    f.write(ini1)
logging.config.fileConfig(tmpini1, disable_existing_loggers=False)
print(logging.getLogger('').level == logging.INFO)  # True
os.remove(tmpini1)

# --- fileConfig: named logger + NullHandler ---
ini2 = """\
[loggers]
keys=root,namedlog

[handlers]
keys=nh2

[formatters]
keys=

[logger_root]
level=WARNING
handlers=

[logger_namedlog]
level=ERROR
handlers=nh2
propagate=0
qualname=namedlog77

[handler_nh2]
class=NullHandler
level=NOTSET
formatter=
args=()
"""
tmpini2 = tempfile.mktemp(suffix='.ini')
with open(tmpini2, 'w') as f:
    f.write(ini2)
logging.config.fileConfig(tmpini2, disable_existing_loggers=False)
nlog = logging.getLogger('namedlog77')
print(nlog.level == logging.ERROR)    # True
print(not nlog.propagate)             # True
os.remove(tmpini2)

# --- listen / stopListening stubs ---
t = logging.config.listen(19876)
print(t is not None)  # True
logging.config.stopListening()
print(True)  # no error

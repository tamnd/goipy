# Intermediate package — imports its own sibling helper, and exposes
# `helper` and `b` as attributes.
from . import helper

GREETING = helper.hello()

"""
This rule will always fire on all reports. It doesn't like them.
"""
from insights.core.plugins import make_fail
from insights.core.plugins import rule


ERROR_KEY = "TUTORIAL_ERROR"


@rule()
def report():
    return make_fail(ERROR_KEY)

# -*- coding: utf-8 -*-

import importlib
import os
import sys
import warnings

os.environ.setdefault("NEW_RELIC_APP_NAME", os.getenv("AWS_LAMBDA_FUNCTION_NAME", ""))
os.environ.setdefault("NEW_RELIC_NO_CONFIG_FILE", "true")
os.environ.setdefault("NEW_RELIC_DISTRIBUTED_TRACING_ENABLED", "true")
os.environ.setdefault("NEW_RELIC_SERVERLESS_MODE_ENABLED", "true")
os.environ.setdefault(
    "NEW_RELIC_TRUSTED_ACCOUNT_KEY", os.getenv("NEW_RELIC_ACCOUNT_ID", "")
)
os.environ.setdefault("NEW_RELIC_PACKAGE_REPORTING_ENABLED", "false")

# The agent will load some environment variables on module import so we need
# to perform the import after setting the necessary environment variables.
import newrelic.agent  # noqa
from newrelic_lambda.lambda_handler import lambda_handler  # noqa

newrelic.agent.initialize()


class IOpipeNoOp(object):
    def __call__(self, *args, **kwargs):
        warnings.warn(
            "Use of context.iopipe.* is no longer supported. "
            "Please see New Relic Python agent documentation here: "
            "https://docs.newrelic.com/docs/agents/python-agent"
        )

    def __getattr__(self, name):
        return IOpipeNoOp()


def get_handler():
    if (
        "NEW_RELIC_LAMBDA_HANDLER" not in os.environ
        or not os.environ["NEW_RELIC_LAMBDA_HANDLER"]
    ):
        raise ValueError(
            "No value specified in NEW_RELIC_LAMBDA_HANDLER environment variable"
        )

    try:
        module_path, handler_name = os.environ["NEW_RELIC_LAMBDA_HANDLER"].rsplit(
            ".", 1
        )
    except ValueError:
        raise ValueError(
            "Improperly formated handler value: %s"
            % os.environ["NEW_RELIC_LAMBDA_HANDLER"]
        )

    try:
        # Use the same check as
        # https://github.com/aws/aws-lambda-python-runtime-interface-client/blob/97dee252434edc56be4cafd54a9af1e7fa041eaf/awslambdaric/bootstrap.py#L33
        if module_path.split(".")[0] in sys.builtin_module_names:
            raise ImportError(
                "Cannot use built-in module %s as a handler module" % module_path
            )

        module = importlib.import_module(module_path.replace("/", "."))
    except ImportError as e:
            raise ImportError("Failed to import module '%s': %s" % (module_path, e))
    except Exception as e:
        raise type(e)(f"Error while importing '{module_path}': {type(e).__name__} {str(e)}").with_traceback(e.__traceback__)
    

    try:
        handler = getattr(module, handler_name)
    except AttributeError:
        raise AttributeError(
            "No handler '%s' in module '%s'" % (handler_name, module_path)
        )

    return handler


# Greedily load the handler during cold start, so we don't pay for it on first invoke
wrapped_handler = get_handler()


@lambda_handler()
def handler(event, context):
    context.iopipe = IOpipeNoOp()
    return wrapped_handler(event, context)

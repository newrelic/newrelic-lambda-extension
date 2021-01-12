package util

import (
	"os"
	"strings"
)

// Detect detects which Lambda runtime we're running in by checking the execution
// environment variable. Should return a value such as "nodejs12.x" or "python3.6", or
// "unknown" if runtime is not known.
//
// see: https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html#configuration-envvars-runtime
func DetectRuntime() string {
	exec_env := os.Getenv("AWS_EXECUTION_ENV")

	if exec_env == "" || !strings.HasPrefix(exec_env, "AWS_Lambda_") {
		return "unknown"
	}

	return strings.TrimPrefix(exec_env, "AWS_Lambda_")
}

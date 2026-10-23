package raijin

type TestError struct {
	Err        error
	StackTrace string
}

type TestResult struct {
	Errors []TestError
}

func (r *TestResult) Failed() bool {
	return len(r.Errors) > 0
}

func (r *TestResult) Add(err error) {
	r.AddWithStackTrace(err, "")
}

func (r *TestResult) AddWithStackTrace(err error, trace string) {
	if err == nil {
		return
	}

	r.Errors = append(r.Errors, TestError{
		Err:        err,
		StackTrace: trace,
	})
}

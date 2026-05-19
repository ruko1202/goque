package periodicprocessor

// WithRunOnStart makes a periodic job enqueue one task when the scheduler starts.
func WithRunOnStart() JobOptions {
	return func(job *Job) {
		job.runOnStart = true
	}
}

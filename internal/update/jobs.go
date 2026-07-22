package update

import "sync"


const (
	StateChecking = "checking"
	StatePulling  = "pulling"
	StateCompose  = "compose"
	StateStarting = "starting"
	StateDone     = "done"
	StateError    = "error"
)


type Job struct {
	Container string `json:"container"`
	State     string `json:"state"`
	Progress  string `json:"progress"`
	Error     string `json:"error,omitempty"`
}

type Jobs struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewJobs() *Jobs {
	return &Jobs{
		jobs: make(map[string]*Job),
	}
}

func (j *Jobs) Set(container, state, progress string) {

	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[container]
	if !ok {
		job = &Job{
			Container: container,
		}
		j.jobs[container] = job
	}

	job.State = state
	job.Progress = progress
	job.Error = ""
}

func (j *Jobs) SetError(container, err string) {

	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[container]
	if !ok {
		job = &Job{
			Container: container,
		}
		j.jobs[container] = job
	}

	job.State = StateError
	job.Error = err
}

func (j *Jobs) Get(container string) (*Job, bool) {

	j.mu.RLock()
	defer j.mu.RUnlock()

	job, ok := j.jobs[container]
	if !ok {
		return nil, false
	}

	copy := *job
	return &copy, true
}

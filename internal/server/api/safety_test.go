package api

import (
	"context"
	"dock/internal/models"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type fakeDocker struct {
	mu         sync.Mutex
	containers map[string]models.Container
	stopped    []string
	failStart  string
	lostHealth bool
	updates    int
	started    chan struct{}
	release    chan struct{}
}

func fixture() *fakeDocker {
	f := &fakeDocker{containers: map[string]models.Container{}}
	for _, name := range requiredServices {
		f.containers[name] = models.Container{ID: name, Name: name, Service: name, Project: name, State: "running", Self: name == "dock"}
	}
	f.containers["app"] = models.Container{ID: "app", Name: "app", Service: "app", Project: "app", State: "running", ImageID: "sha256:old"}
	return f
}
func (f *fakeDocker) Container(_ context.Context, id string) (models.Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id == "alias" {
		id = "app"
	}
	c, ok := f.containers[id]
	if !ok {
		return c, errors.New("not found")
	}
	if f.lostHealth && len(f.stopped) > 0 && id == "ntfy" {
		c.HasHealthcheck = true
		c.Health = "unhealthy"
	}
	return c, nil
}
func (f *fakeDocker) ListContainers(context.Context) ([]models.Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := []models.Container{}
	for _, c := range f.containers {
		result = append(result, c)
	}
	return result, nil
}
func (f *fakeDocker) StartContainer(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id == f.failStart {
		return errors.New("start failed")
	}
	c := f.containers[id]
	c.State = "running"
	f.containers[id] = c
	return nil
}
func (f *fakeDocker) StopContainer(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, id)
	c := f.containers[id]
	c.State = "exited"
	f.containers[id] = c
	return nil
}
func (f *fakeDocker) RestartContainer(context.Context, string) error            { return nil }
func (f *fakeDocker) ComposeWorkingDir(context.Context, string) (string, error) { return "", nil }
func (f *fakeDocker) ImageDigest(context.Context, string) (string, error)       { return "", nil }
func (f *fakeDocker) ImageInfo(context.Context, string) (models.Image, error) {
	return models.Image{}, nil
}
func (f *fakeDocker) ContainerLogs(context.Context, string, int) (io.ReadCloser, error) {
	return nil, nil
}
func (f *fakeDocker) ComposeUpdate(context.Context, string) error {
	f.mu.Lock()
	f.updates++
	f.mu.Unlock()
	if f.started != nil {
		f.started <- struct{}{}
	}
	if f.release != nil {
		<-f.release
	}
	return nil
}
func emergency(t *testing.T, f *fakeDocker) *httptest.ResponseRecorder {
	t.Helper()
	h := NewEmergencyHandler(f)
	h.healthTimeout = 5 * time.Millisecond
	w := httptest.NewRecorder()
	h.Activate(w, httptest.NewRequest("POST", "/api/emergency", nil))
	return w
}
func TestEmergencyRequiresAllFourBeforeStopping(t *testing.T) {
	for _, scenario := range []string{"missing", "unhealthy", "start-failed", "starting-health"} {
		t.Run(scenario, func(t *testing.T) {
			f := fixture()
			c := f.containers["ntfy"]
			switch scenario {
			case "missing":
				delete(f.containers, "ntfy")
			case "unhealthy":
				c.HasHealthcheck = true
				c.Health = "unhealthy"
				f.containers["ntfy"] = c
			case "starting-health":
				c.HasHealthcheck = true
				c.Health = "starting"
				f.containers["ntfy"] = c
			case "start-failed":
				c.State = "exited"
				f.containers["ntfy"] = c
				f.failStart = "ntfy"
			}
			w := emergency(t, f)
			if w.Code != 409 || len(f.stopped) != 0 {
				t.Fatalf("code=%d stopped=%v body=%s", w.Code, f.stopped, w.Body.String())
			}
		})
	}
}
func TestEmergencyStartsStoppedEssentialAndPreservesRenamedDock(t *testing.T) {
	f := fixture()
	c := f.containers["dock"]
	c.Name = "5602aeff8013_dock"
	f.containers["dock"] = c
	c = f.containers["cloudflared"]
	c.State = "exited"
	f.containers["cloudflared"] = c
	w := emergency(t, f)
	if w.Code != 200 || len(f.stopped) != 1 || f.stopped[0] != "app" {
		t.Fatal(w.Body.String(), f.stopped)
	}
	if f.containers["cloudflared"].State != "running" {
		t.Fatal("essential not started")
	}
}
func TestEmergencyStopsPausedAndRestarting(t *testing.T) {
	for _, state := range []string{"paused", "restarting"} {
		t.Run(state, func(t *testing.T) {
			f := fixture()
			c := f.containers["app"]
			c.State = state
			f.containers["app"] = c
			w := emergency(t, f)
			if w.Code != 200 || len(f.stopped) != 1 {
				t.Fatal(w.Body.String(), f.stopped)
			}
		})
	}
}
func TestEmergencyDoesNotStopAnythingWhenEssentialAmbiguous(t *testing.T) {
	f := fixture()
	f.containers["other"] = models.Container{ID: "other", Name: "other", Service: "ntfy", State: "running"}
	w := emergency(t, f)
	if w.Code != 409 || len(f.stopped) != 0 {
		t.Fatal(w.Body.String())
	}
}
func TestEmergencyRechecksHealthAndDoesNotReverse(t *testing.T) {
	f := fixture()
	f.lostHealth = true
	f.containers["app2"] = models.Container{ID: "app2", Name: "app2", State: "running"}
	w := emergency(t, f)
	if w.Code != 409 || len(f.stopped) != 1 {
		t.Fatal(w.Body.String(), f.stopped)
	}
	if f.containers[f.stopped[0]].State != "exited" {
		t.Fatal("unexpected automatic reversal")
	}
}
func TestEmergencyFinishesDespiteRequestCancellation(t *testing.T) {
	f := fixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	NewEmergencyHandler(f).Activate(w, httptest.NewRequest("POST", "/", nil).WithContext(ctx))
	if w.Code != 200 || len(f.stopped) != 1 {
		t.Fatal(w.Body.String())
	}
}
func request(id, job string) *http.Request {
	r := httptest.NewRequest("POST", "/api/update/"+id+"?job="+job, nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
}
func TestConcurrentAliasesAndRequestRetriesCreateOneJob(t *testing.T) {
	f := fixture()
	f.started = make(chan struct{}, 100)
	f.release = make(chan struct{})
	gate := NewOperations()
	h := NewUpdateHandler(f, gate)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "app"
			if i%2 == 0 {
				id = "alias"
			}
			w := httptest.NewRecorder()
			h.Update(w, request(id, "same-request-123"))
			if w.Code != 200 && w.Code != 202 {
				t.Errorf("code=%d %s", w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()
	<-f.started
	f.mu.Lock()
	calls := f.updates
	f.mu.Unlock()
	if calls != 1 {
		t.Fatal(calls)
	}
	w := httptest.NewRecorder()
	gate.Guard(func(w http.ResponseWriter, r *http.Request) { t.Error("concurrent mutation allowed") })(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	close(f.release)
}
func TestSelfUpdateRejectedBeforeMutation(t *testing.T) {
	f := fixture()
	h := NewUpdateHandler(f, NewOperations())
	w := httptest.NewRecorder()
	h.Update(w, request("dock", "self-request-123"))
	if w.Code != 409 || f.updates != 0 {
		t.Fatal(w.Code)
	}
}
func TestJobStatusUsesStableRequestIDAfterContainerRecreation(t *testing.T) {
	f := fixture()
	f.started = make(chan struct{}, 1)
	f.release = make(chan struct{})
	h := NewUpdateHandler(f, NewOperations())
	w := httptest.NewRecorder()
	h.Update(w, request("app", "stable-request-123"))
	<-f.started
	f.mu.Lock()
	delete(f.containers, "app")
	f.mu.Unlock()
	w = httptest.NewRecorder()
	h.JobStatus(w, request("app", "stable-request-123"))
	var job UpdateJob
	json.Unmarshal(w.Body.Bytes(), &job)
	if job.Status != "running" {
		t.Fatal(w.Body.String())
	}
	close(f.release)
}
func TestUnknownJobCannotPretendCompleted(t *testing.T) {
	h := NewUpdateHandler(fixture(), NewOperations())
	w := httptest.NewRecorder()
	h.JobStatus(w, request("app", "missing-request-123"))
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["status"] != "unknown" {
		t.Fatal(result)
	}
}

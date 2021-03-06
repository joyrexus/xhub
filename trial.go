package xhub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/joyrexus/buckets"
	"github.com/julienschmidt/httprouter"
)

// NewTrialController initializes a new instance of our trial controller.
func NewTrialController(host string, bux *buckets.DB) *TrialController {
	// Create/open bucket for storing study-related data.
	studies, err := bux.New([]byte("studies"))
	if err != nil {
		log.Fatalf("couldn't create/open studies bucket: %v\n", err)
	}
	return &TrialController{host, studies}
}

// A TrialController handles requests for trial resources.
type TrialController struct {
	host    string
	studies *buckets.Bucket
}

// Post handles POST requests for `/studies/:study/trials`, storing
// the trial data sent.
func (c *TrialController) Post(w http.ResponseWriter, r *http.Request,
	_ httprouter.Params) {

	var trial Resource
	err := json.NewDecoder(r.Body).Decode(&trial)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	key := []byte(trial.ID)
	if err := c.studies.Put(key, trial.Data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// List handles GET requests for `/studies/:study/trials`, returning a list
// of available trials for a particular study.
func (c *TrialController) List(w http.ResponseWriter, r *http.Request,
	p httprouter.Params) {

	study := p.ByName("study")
	prefix := fmt.Sprintf("/studies/%s/trials", study)
	items, err := c.studies.PrefixItems([]byte(prefix))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	resources := []*Resource{}

	// Append each item to the list of resources.
	for _, trial := range items {
		id := string(trial.Key)
		url := "http://" + c.host + id
		rsc := &Resource{
			Version: "1",
			Type:    "trial",
			ID:      id,
			URL:     url,
			Data:    trial.Value,
		}
		resources = append(resources, rsc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// Get handles GET requests for `/studies/:study/trials/:trial`, returning
// the raw json data payload for the requested trial.
func (c *TrialController) Get(w http.ResponseWriter, r *http.Request,
	p httprouter.Params) {

	study, trial := p.ByName("study"), p.ByName("trial")
	id := fmt.Sprintf("/studies/%s/trials/%s", study, trial)
	data, err := c.studies.Get([]byte(id))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if data == nil {
		http.Error(w, id+" not found", http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// Delete handles DELETE requests for `/studies/:study/trials/:trial`,
// deleting all items from the studies bucket that are associated with
// the specified study and trial.
func (c *TrialController) Delete(w http.ResponseWriter, r *http.Request,
	p httprouter.Params) {

	study, trial := p.ByName("study"), p.ByName("trial")

	// delete all items with these study + trial prefixes
	for _, pre := range []string{
		fmt.Sprintf("/studies/%s/trials/%s", study, trial),
		fmt.Sprintf("/files/%s/%s", study, trial),
	} {
		items, err := c.studies.PrefixItems([]byte(pre))
		if err != nil {
			e := fmt.Sprintf("couldn't retrieve items with prefix %q: %v",
				pre,
				err,
			)
			http.Error(w, e, 500)
			return
		}
		for _, item := range items {
			if err := c.studies.Delete(item.Key); err != nil {
				e := fmt.Sprintf("couldn't delete item %q: %v", item.Key, err)
				http.Error(w, e, 500)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

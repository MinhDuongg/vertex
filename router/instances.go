package router

import (
	"io"

	"github.com/gin-contrib/sse"
	"github.com/vertex-center/vertex/pkg/log"
	"github.com/vertex-center/vertex/pkg/router"
	"github.com/vertex-center/vertex/types"
	"github.com/vertex-center/vertex/types/api"
)

func addInstancesRoutes(r *router.Group) {
	r.GET("", handleGetInstances)
	r.GET("/search", handleSearchInstances)
	r.GET("/checkupdates", handleCheckForUpdates)
	r.GET("/events", headersSSE, handleInstancesEvents)
}

// handleGetInstances returns all installed instances.
func handleGetInstances(c *router.Context) {
	installed := instanceService.GetAll()
	c.JSON(installed)
}

// handleSearchInstances returns all installed instances that match the query.
func handleSearchInstances(c *router.Context) {
	query := types.InstanceQuery{}

	features := c.QueryArray("features[]")
	if len(features) > 0 {
		query.Features = features
	}

	installed := instanceService.Search(query)
	c.JSON(installed)
}

// handleCheckForUpdates checks for updates for all installed instances.
// Errors can be:
//   - check_for_updates_failed
func handleCheckForUpdates(c *router.Context) {
	instances, err := instanceService.CheckForUpdates()
	if err != nil {
		c.Abort(router.Error{
			Code:           api.ErrFailedToCheckForUpdates,
			PublicMessage:  "Failed to check for updates.",
			PrivateMessage: err.Error(),
		})
		return
	}

	c.JSON(instances)
}

// handleInstancesEvents returns a stream of events related to instances.
func handleInstancesEvents(c *router.Context) {
	eventsChan := make(chan sse.Event)
	defer close(eventsChan)

	done := c.Request.Context().Done()

	listener := types.NewTempListener(func(e interface{}) {
		switch e.(type) {
		case types.EventInstancesChange:
			eventsChan <- sse.Event{
				Event: types.EventNameInstancesChange,
			}
		}
	})

	eventInMemoryAdapter.AddListener(listener)
	defer eventInMemoryAdapter.RemoveListener(listener)

	first := true

	c.Stream(func(w io.Writer) bool {
		if first {
			err := sse.Encode(w, sse.Event{
				Event: "open",
			})

			if err != nil {
				log.Error(err)
				return false
			}
			first = false
			return true
		}

		select {
		case e := <-eventsChan:
			err := sse.Encode(w, e)
			if err != nil {
				log.Error(err)
			}
			return true
		case <-done:
			return false
		}
	})
}

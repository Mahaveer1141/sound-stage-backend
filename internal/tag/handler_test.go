package tag

import (
	"net/http"
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockTagService struct{ mock.Mock }

func (m *mockTagService) Create(input *CreateTagParams) (*Tag, error) {
	args := m.Called(input)
	t, _ := args.Get(0).(*Tag)
	return t, args.Error(1)
}
func (m *mockTagService) List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, int64, error) {
	args := m.Called(filter, sort, p)
	tags, _ := args.Get(0).([]Tag)
	return tags, args.Get(1).(int64), args.Error(2)
}

type handlerHarness struct {
	svc     *mockTagService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockTagService)
	return &handlerHarness{svc: svc, handler: NewHandler(svc)}
}

func TestHandler_Create(t *testing.T) {
	t.Run("success: creates tag and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		created := &Tag{Name: "live"}
		h.svc.On("Create", mock.MatchedBy(func(in *CreateTagParams) bool {
			return in.Name == "live"
		})).Return(created, nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/tags", CreateTagParams{Name: "live"})

		h.handler.Create(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: malformed JSON body returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/tags", "{not json")

		h.handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: empty name fails validation with 422", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/tags", CreateTagParams{Name: ""})

		h.handler.Create(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Create", mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/tags", CreateTagParams{Name: "live"})

		h.handler.Create(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}
func TestHandler_List(t *testing.T) {
	t.Run("success: returns 200 with paginated tags", func(t *testing.T) {
		h := newHandlerHarness(t)
		tags := []Tag{{Name: "live"}, {Name: "jazz"}}
		h.svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(tags, int64(2), nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/tags?page=1&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("success: forwards search query and sort params", func(t *testing.T) {
		h := newHandlerHarness(t)
		wantFilter := TagFilter{Query: "ja"}
		wantSort := listopts.Sort{Field: "name", Order: "asc"}
		wantP := listopts.Pagination{Page: 1, PageSize: 25}
		tags := []Tag{{Name: "jazz"}}
		h.svc.On("List", wantFilter, wantSort, wantP).Return(tags, int64(1), nil)

		w, c := testutil.NewTestContext(
			http.MethodGet,
			"/tags?page=1&pageSize=25&query=ja&field=name&order=asc",
			nil,
		)

		h.handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-positive page returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/tags?page=0&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/tags?page=1&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

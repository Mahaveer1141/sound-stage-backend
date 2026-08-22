package category

import (
	"net/http"
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockCategoryService struct{ mock.Mock }

func (m *mockCategoryService) List() ([]Category, error) {
	args := m.Called()
	categories, _ := args.Get(0).([]Category)
	return categories, args.Error(1)
}

type handlerHarness struct {
	svc     *mockCategoryService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockCategoryService)
	return &handlerHarness{svc: svc, handler: NewHandler(svc)}
}

func TestHandler_List(t *testing.T) {
	t.Run("success: returns 200 with categories", func(t *testing.T) {
		h := newHandlerHarness(t)
		categories := []Category{{Name: "Gaming"}, {Name: "Music & Audio"}}
		h.svc.On("List", mock.Anything, mock.Anything).Return(categories, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/categories", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("List", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/categories", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

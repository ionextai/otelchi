package respwriter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet_DefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Get(rec)
	defer Put(rw)

	_, err := rw.ResponseWriter.Write([]byte("hello"))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rw.StatusCode)
	assert.Equal(t, int64(5), rw.BytesWritten)
}

func TestGet_ExplicitWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Get(rec)
	defer Put(rw)

	rw.ResponseWriter.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.StatusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGet_AccumulatesBytesAcrossWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Get(rec)
	defer Put(rw)

	_, err := rw.ResponseWriter.Write([]byte("foo"))
	assert.NoError(t, err)
	_, err = rw.ResponseWriter.Write([]byte("barbaz"))
	assert.NoError(t, err)

	assert.Equal(t, int64(9), rw.BytesWritten)
}

func TestGet_SuperfluousWriteHeaderIsNoOp(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := Get(rec)
	defer Put(rw)

	rw.ResponseWriter.WriteHeader(http.StatusAccepted)
	rw.ResponseWriter.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, http.StatusAccepted, rw.StatusCode)
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestGet_PoolReuseDoesNotLeakState(t *testing.T) {
	rec1 := httptest.NewRecorder()
	rw1 := Get(rec1)
	rw1.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	_, err := rw1.ResponseWriter.Write([]byte("failure"))
	assert.NoError(t, err)
	Put(rw1)

	rec2 := httptest.NewRecorder()
	rw2 := Get(rec2)
	defer Put(rw2)

	assert.Equal(t, http.StatusOK, rw2.StatusCode)
	assert.Equal(t, int64(0), rw2.BytesWritten)
}

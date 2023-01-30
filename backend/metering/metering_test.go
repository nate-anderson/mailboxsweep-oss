package metering

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"mailboxsweep"
	"mailboxsweep/assert"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"time"
)

func TestMetering(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	testAddr := "admin3@example.org"
	repo := NewUsageRepository(db, table)

	dialer := NewDialer(ctx, testAddr, repo)
	var conn *Conn

	dial := func(network string, addr string) (net.Conn, error) {
		goConn, err := dialer.Dial(network, addr)
		if meteredConn, ok := goConn.(*Conn); ok {
			conn = meteredConn
		} else {
			t.Fatal("dialer returned something other than *Conn")
		}
		return goConn, err
	}

	tr := &http.Transport{
		Dial:                dial,
		TLSHandshakeTimeout: 2 * time.Second,
	}

	client := http.Client{Transport: tr}

	var reqSize int64
	testHandler := func(w http.ResponseWriter, req *http.Request) {
		b, err := httputil.DumpRequest(req, true)
		if err != nil {
			t.Fatal(err)
		}
		reqSize = int64(len(b))
	}

	srv := httptest.NewServer(http.HandlerFunc(testHandler))
	defer srv.Close()

	t.Run("send content", func(t *testing.T) {
		n := 10000
		randBytes := make([]byte, n)
		_, err := io.ReadFull(rand.Reader, randBytes)
		assert.ThatError(t, err).IsNil()

		bodyReader := bytes.NewReader(randBytes)
		req, err := http.NewRequest(http.MethodPost, srv.URL, bodyReader)
		assert.ThatError(t, err).IsNil()

		res, err := client.Do(req)
		assert.ThatError(t, err).IsNil()
		assert.That(t, res.StatusCode).Equals(http.StatusOK)
		assert.That(t, conn).NotEqual(nil)
		resDump, err := httputil.DumpResponse(res, true)
		assert.ThatError(t, err).IsNil()
		resSize := len(resDump)

		// it seems hard to test this precisely
		// just test that the difference is < 2% for now, we undershoot the Gmail quota already
		// so there's no way to violate it with <2% delta
		actualReadWrite := int(reqSize) + int(resSize)
		diff := math.Abs(float64(conn.bytesTransfered - actualReadWrite))
		deltaPercent := diff / float64(actualReadWrite)
		assert.ThatOrdered(t, deltaPercent).LessThanOrEqual(0.02)
	})
}

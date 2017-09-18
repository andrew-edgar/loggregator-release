package api_test

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	sharedapi "tools/reliability/api"
	"tools/reliability/server/internal/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateTestHandler", func() {
	It("passes the test to a runner", func() {
		runner := &spyRunner{}
		h := api.NewCreateTestHandler(runner)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, &http.Request{
			Method: "POST",
			Body: &requestBody{
				Reader: strings.NewReader(`{"cycles": 1000, "delay":"1s", "timeout":"60s"}`),
			},
		})

		Expect(recorder.Code).To(Equal(http.StatusCreated))
		Eventually(runner.called).Should(Equal(int64(1)))
	})

	It("responds with the created test", func() {
		runner := &spyRunner{}
		h := api.NewCreateTestHandler(runner)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, &http.Request{
			Method: "POST",
			Body: &requestBody{
				Reader: strings.NewReader(`{"cycles": 1000, "delay":"1s", "timeout":"60s"}`),
			},
		})

		var test sharedapi.Test
		err := json.Unmarshal([]byte(recorder.Body.String()), &test)
		Expect(err).ToNot(HaveOccurred())
		Expect(test.ID).ToNot(Equal(int64(0)))
		Expect(test.Cycles).To(Equal(uint64(1000)))
		Expect(test.Delay).To(Equal(
			sharedapi.Duration(
				int64(1000000000),
			),
		))
		Expect(test.Timeout).To(Equal(
			sharedapi.Duration(
				int64(60000000000),
			),
		))
		Expect(test.StartTime).ToNot(Equal(int64(0)))
	})

	DescribeTable("with an invalid test request", func(body string) {
		runner := &spyRunner{}
		h := api.NewCreateTestHandler(runner)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, &http.Request{
			Method: "POST",
			Body: &requestBody{
				Reader: strings.NewReader(body),
			},
		})
		_, err := ioutil.ReadAll(recorder.Body)

		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("empty", `{}`),
		Entry("without timeout", `{"cycles": 1, "delay": "1s"}`),
		Entry("with invalid timeout", `{"cycles": 1, "timeout": "one second"}`),
		Entry("with malformed json", `!#$^?!#$^`),
	)

	It("returns MethodNotAllowed on anything but a POST", func() {
		runner := &spyRunner{}
		h := api.NewCreateTestHandler(runner)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, &http.Request{
			Method: "GET",
		})

		Expect(recorder.Code).To(Equal(http.StatusMethodNotAllowed))
	})

	Context("with an error returned from the runner", func() {
		It("responds with a 500", func() {
			runner := &spyRunner{
				err: errors.New("some-error"),
			}
			h := api.NewCreateTestHandler(runner)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, &http.Request{
				Method: "POST",
				Body: &requestBody{
					Reader: strings.NewReader(`{"cycles": 1000, "delay":"1s", "timeout":"60s"}`),
				},
			})

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			body, err := ioutil.ReadAll(recorder.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(Equal([]byte("some-error")))
			Eventually(runner.called).Should(Equal(int64(1)))
		})
	})
})

type spyRunner struct {
	called int64
	err    error
}

func (s *spyRunner) Run(*sharedapi.Test) error {
	s.called++
	return s.err
}

type requestBody struct {
	io.Reader
	Closer io.Closer
}

func (r *requestBody) Close() error {
	return nil
}

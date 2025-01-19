package circuit

import (
	"fmt"
	"net/http"
)

func SendHttpRequest(req *http.Request, clients ...*http.Client) (*http.Response, error) {

	var client = http.DefaultClient
	if len(clients) > 0 {
		client = clients[0]
	}

	baseUrl := fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)

	cb, ok := circuit[baseUrl]
	if !ok {
		cb = newCircuitBreaker(baseUrl)
		circuit[baseUrl] = cb
	}

	var (
		response *http.Response
	)

	err := cb.Execute(func() error {

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		response = resp

		return nil
	})

	if err != nil && configCircuit.maxRetry > 0 {
		for i := 0; i < configCircuit.maxRetry; i++ {
			err = cb.Execute(func() error {

				resp, err := client.Do(req)
				if err != nil {
					return err
				}

				if resp.StatusCode >= http.StatusInternalServerError {
					return ERR_CIRCUIT_INTERAL_SERVER_ERROR
				}

				response = resp

				return nil
			})

			if err == nil {
				break
			}
		}
	}

	return response, err

}

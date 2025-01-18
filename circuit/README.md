# Circuit Breake Gowok

this is plugin for setup cicuit breaker on gowok

example config:
``` yaml
circuitBreaker:
    # duration for circuit breaker
    # format
    # s = second
    # m = minute
    # h = hour
    # d = day
    duration: 4m20s
    # this for http client request
    maxRetry: 3
    # circuit open if more than max failure
    maxFailure: 10
```

## How to use
Configure
``` go
import (
    "github.com/gowok/gowok"
    "github.com/gowok/plugins/circuit"
)

func main() {
    project = gowok.Get()

    project.Configure(
        circuit.Configure,
    )

    project.Run()
}

```


using on code block
```go
import "github.com/gowok/plugins/circuit"

func (s *service)  GetPromoBySegment(ctx context.Context, segment string) ([]models.Promo, error) {

    var results []models.Promo

    err := circuit.Get("getPromoBySegment").Execute(func()error {

        promos, err := s.repo.GetPromoBySegment(ctx, segment)
        if err != nil {
            return err
        }

        results = promos
        return nil
    })

    return results, err
}

```


using for http request

``` go
import (
    "net/http"

    "github.com/born2ngopi/learn/models"

    "github.com/gowok/plugin/circuit"
)

func GetUsers() (models.User, error) {

    req, err := http.NewRequest(http.MethodGet, "https://github.com/born2ngopi", nil)
    // handle error

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(constants.USER_SERVICE_TOKEN)))

    res, err := circuit.SendHttpRequest(req, http.DefaultClient)
    if err != nil {
        if errors.Is(err, circuit.ERR_CIRCUIT_OPEN) {
            // posible server down
            return models.User{}, fmt.Errorf("github is down")
        }
    }

    // do logic ...

}

```
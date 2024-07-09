package grifts

import (
	"mc-iam-manager/actions"

	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
	//나중에 default bind setting 을 위해 처리
	// binding.Register("application/json", func(r *http.Request, resp *http.Response, interface{}) error {
	// 	b, err := io.ReadAll(r.Body)
	// 	t, t_err := io.ReadAll(resp.Body)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return json.Unmarshal(b, i)
	// })
}

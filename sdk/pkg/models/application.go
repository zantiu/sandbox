package models

import (
	"encoding/json"

	"github.com/margo/dev-repo/sdk/utils"
)

type Application struct {
	AppId string
	AppPkgOp
	AppPkgOpStatus
	ApplicationDescription
}

func NewApplication(desc ApplicationDescription, op AppPkgOp, status AppPkgOpStatus, details string) Application {
	return Application{
		AppId:          utils.GenerateAppId(),
		AppPkgOpStatus: status,
	}
}

func ParseApplicationFromBytes(data []byte) (Application, error) {
	app := Application{}
	if err := json.Unmarshal(data, &app); err != nil {
		return app, err
	}
	return Application{}, nil
}

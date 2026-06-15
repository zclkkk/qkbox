package singboxadapter

import (
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/zclkkk/qkbox/shared/model"
)

func Validate(configJSON string) model.Diagnostics {
	if err := libbox.CheckConfig(configJSON); err != nil {
		return model.Diagnostics{
			Status: model.ValidationStatusInvalid,
			Entries: []model.ValidationDiagnostic{
				{
					Severity: model.SeverityError,
					Message:  err.Error(),
				},
			},
		}
	}
	return model.Diagnostics{Status: model.ValidationStatusValid}
}

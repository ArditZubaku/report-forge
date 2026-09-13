package reports

import "uuid"

type SQSMessage struct {
	UserId   uuid.UUID `json:"userId"`
	ReportId uuid.UUID `json:"reportId"`
}

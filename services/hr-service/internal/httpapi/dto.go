package httpapi

import (
	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type leaveRequestRecord struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	EmployeeID     uuid.UUID  `json:"employeeId"`
	LeaveType      string     `json:"leaveType"`
	StartDate      string     `json:"startDate"`
	EndDate        string     `json:"endDate"`
	DaysCount      int        `json:"daysCount"`
	Reason         string     `json:"reason,omitempty"`
	Status         string     `json:"status"`
	CreatedBy      uuid.UUID  `json:"createdBy"`
	ApprovedBy     *uuid.UUID `json:"approvedBy,omitempty"`
	CreatedAt      string     `json:"createdAt"`
}

type attendanceLogRecord struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	EmployeeID     uuid.UUID `json:"employeeId"`
	LogDate        string    `json:"logDate"`
	Status         string    `json:"status"`
	ClockIn        string    `json:"clockIn,omitempty"`
	ClockOut       string    `json:"clockOut,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedBy      uuid.UUID `json:"createdBy"`
}

type employeeMutationReq struct {
	EmployeeNo  string `json:"employeeNo"`
	FullName    string `json:"fullName"`
	NationalID  string `json:"nationalId"`
	DateOfBirth string `json:"dateOfBirth"`
	Email       string `json:"email"`
	HireDate    string `json:"hireDate"`
	UserRefID   string `json:"userRefId"`
}

func hrOrgInScope(scope []uuid.UUID, target uuid.UUID) bool {
	if target == platform.SystemOrgID {
		return false
	}
	for _, id := range scope {
		if id == target {
			return true
		}
	}
	return false
}

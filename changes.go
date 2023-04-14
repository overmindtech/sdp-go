package sdp

import (
	"github.com/google/uuid"
)

func (x *OnboardingProperties) GetAppUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetAppUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *OnboardingProperties) GetAwsSourceUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetAwsSourceUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *OnboardingProperties) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

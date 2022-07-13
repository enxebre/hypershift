// Code generated by go-swagger; DO NOT EDIT.

package p_cloud_volume_onboarding

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/IBM-Cloud/power-go-client/power/models"
)

// PcloudVolumeOnboardingGetReader is a Reader for the PcloudVolumeOnboardingGet structure.
type PcloudVolumeOnboardingGetReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PcloudVolumeOnboardingGetReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPcloudVolumeOnboardingGetOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewPcloudVolumeOnboardingGetBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 401:
		result := NewPcloudVolumeOnboardingGetUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewPcloudVolumeOnboardingGetNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewPcloudVolumeOnboardingGetInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPcloudVolumeOnboardingGetOK creates a PcloudVolumeOnboardingGetOK with default headers values
func NewPcloudVolumeOnboardingGetOK() *PcloudVolumeOnboardingGetOK {
	return &PcloudVolumeOnboardingGetOK{}
}

/* PcloudVolumeOnboardingGetOK describes a response with status code 200, with default header values.

OK
*/
type PcloudVolumeOnboardingGetOK struct {
	Payload models.VolumeOnboardings
}

func (o *PcloudVolumeOnboardingGetOK) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/volumes/onboarding/{volume_onboarding_id}][%d] pcloudVolumeOnboardingGetOK  %+v", 200, o.Payload)
}
func (o *PcloudVolumeOnboardingGetOK) GetPayload() models.VolumeOnboardings {
	return o.Payload
}

func (o *PcloudVolumeOnboardingGetOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudVolumeOnboardingGetBadRequest creates a PcloudVolumeOnboardingGetBadRequest with default headers values
func NewPcloudVolumeOnboardingGetBadRequest() *PcloudVolumeOnboardingGetBadRequest {
	return &PcloudVolumeOnboardingGetBadRequest{}
}

/* PcloudVolumeOnboardingGetBadRequest describes a response with status code 400, with default header values.

Bad Request
*/
type PcloudVolumeOnboardingGetBadRequest struct {
	Payload *models.Error
}

func (o *PcloudVolumeOnboardingGetBadRequest) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/volumes/onboarding/{volume_onboarding_id}][%d] pcloudVolumeOnboardingGetBadRequest  %+v", 400, o.Payload)
}
func (o *PcloudVolumeOnboardingGetBadRequest) GetPayload() *models.Error {
	return o.Payload
}

func (o *PcloudVolumeOnboardingGetBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudVolumeOnboardingGetUnauthorized creates a PcloudVolumeOnboardingGetUnauthorized with default headers values
func NewPcloudVolumeOnboardingGetUnauthorized() *PcloudVolumeOnboardingGetUnauthorized {
	return &PcloudVolumeOnboardingGetUnauthorized{}
}

/* PcloudVolumeOnboardingGetUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type PcloudVolumeOnboardingGetUnauthorized struct {
	Payload *models.Error
}

func (o *PcloudVolumeOnboardingGetUnauthorized) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/volumes/onboarding/{volume_onboarding_id}][%d] pcloudVolumeOnboardingGetUnauthorized  %+v", 401, o.Payload)
}
func (o *PcloudVolumeOnboardingGetUnauthorized) GetPayload() *models.Error {
	return o.Payload
}

func (o *PcloudVolumeOnboardingGetUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudVolumeOnboardingGetNotFound creates a PcloudVolumeOnboardingGetNotFound with default headers values
func NewPcloudVolumeOnboardingGetNotFound() *PcloudVolumeOnboardingGetNotFound {
	return &PcloudVolumeOnboardingGetNotFound{}
}

/* PcloudVolumeOnboardingGetNotFound describes a response with status code 404, with default header values.

Not Found
*/
type PcloudVolumeOnboardingGetNotFound struct {
	Payload *models.Error
}

func (o *PcloudVolumeOnboardingGetNotFound) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/volumes/onboarding/{volume_onboarding_id}][%d] pcloudVolumeOnboardingGetNotFound  %+v", 404, o.Payload)
}
func (o *PcloudVolumeOnboardingGetNotFound) GetPayload() *models.Error {
	return o.Payload
}

func (o *PcloudVolumeOnboardingGetNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudVolumeOnboardingGetInternalServerError creates a PcloudVolumeOnboardingGetInternalServerError with default headers values
func NewPcloudVolumeOnboardingGetInternalServerError() *PcloudVolumeOnboardingGetInternalServerError {
	return &PcloudVolumeOnboardingGetInternalServerError{}
}

/* PcloudVolumeOnboardingGetInternalServerError describes a response with status code 500, with default header values.

Internal Server Error
*/
type PcloudVolumeOnboardingGetInternalServerError struct {
	Payload *models.Error
}

func (o *PcloudVolumeOnboardingGetInternalServerError) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/volumes/onboarding/{volume_onboarding_id}][%d] pcloudVolumeOnboardingGetInternalServerError  %+v", 500, o.Payload)
}
func (o *PcloudVolumeOnboardingGetInternalServerError) GetPayload() *models.Error {
	return o.Payload
}

func (o *PcloudVolumeOnboardingGetInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

package index

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/danisla/esio/models"
)

/*GetStartEndOK All indices in [start,end] range are availble and ready.

swagger:response getStartEndOK
*/
type GetStartEndOK struct {

	// In: body
	Payload *models.Ready `json:"body,omitempty"`
}

// NewGetStartEndOK creates GetStartEndOK with default headers values
func NewGetStartEndOK() *GetStartEndOK {
	return &GetStartEndOK{}
}

// WithPayload adds the payload to the get start end o k response
func (o *GetStartEndOK) WithPayload(payload *models.Ready) *GetStartEndOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end o k response
func (o *GetStartEndOK) SetPayload(payload *models.Ready) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*GetStartEndPartialContent Zero or more of the indices in the [start,end] range are available and ready.

swagger:response getStartEndPartialContent
*/
type GetStartEndPartialContent struct {

	// In: body
	Payload *models.Partial `json:"body,omitempty"`
}

// NewGetStartEndPartialContent creates GetStartEndPartialContent with default headers values
func NewGetStartEndPartialContent() *GetStartEndPartialContent {
	return &GetStartEndPartialContent{}
}

// WithPayload adds the payload to the get start end partial content response
func (o *GetStartEndPartialContent) WithPayload(payload *models.Partial) *GetStartEndPartialContent {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end partial content response
func (o *GetStartEndPartialContent) SetPayload(payload *models.Partial) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndPartialContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(206)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*GetStartEndBadRequest invalid time range provided

swagger:response getStartEndBadRequest
*/
type GetStartEndBadRequest struct {

	// In: body
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetStartEndBadRequest creates GetStartEndBadRequest with default headers values
func NewGetStartEndBadRequest() *GetStartEndBadRequest {
	return &GetStartEndBadRequest{}
}

// WithPayload adds the payload to the get start end bad request response
func (o *GetStartEndBadRequest) WithPayload(payload *models.Error) *GetStartEndBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end bad request response
func (o *GetStartEndBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*GetStartEndNotFound Indices in the [start,end] range are available for restore but not available.

swagger:response getStartEndNotFound
*/
type GetStartEndNotFound struct {

	// In: body
	Payload *models.NotReady `json:"body,omitempty"`
}

// NewGetStartEndNotFound creates GetStartEndNotFound with default headers values
func NewGetStartEndNotFound() *GetStartEndNotFound {
	return &GetStartEndNotFound{}
}

// WithPayload adds the payload to the get start end not found response
func (o *GetStartEndNotFound) WithPayload(payload *models.NotReady) *GetStartEndNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end not found response
func (o *GetStartEndNotFound) SetPayload(payload *models.NotReady) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*GetStartEndRequestRangeNotSatisfiable No indices are available for restore in given [start,end] range.

swagger:response getStartEndRequestRangeNotSatisfiable
*/
type GetStartEndRequestRangeNotSatisfiable struct {

	// In: body
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetStartEndRequestRangeNotSatisfiable creates GetStartEndRequestRangeNotSatisfiable with default headers values
func NewGetStartEndRequestRangeNotSatisfiable() *GetStartEndRequestRangeNotSatisfiable {
	return &GetStartEndRequestRangeNotSatisfiable{}
}

// WithPayload adds the payload to the get start end request range not satisfiable response
func (o *GetStartEndRequestRangeNotSatisfiable) WithPayload(payload *models.Error) *GetStartEndRequestRangeNotSatisfiable {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end request range not satisfiable response
func (o *GetStartEndRequestRangeNotSatisfiable) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndRequestRangeNotSatisfiable) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(416)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*GetStartEndDefault Unexpected error

swagger:response getStartEndDefault
*/
type GetStartEndDefault struct {
	_statusCode int

	// In: body
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetStartEndDefault creates GetStartEndDefault with default headers values
func NewGetStartEndDefault(code int) *GetStartEndDefault {
	if code <= 0 {
		code = 500
	}

	return &GetStartEndDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the get start end default response
func (o *GetStartEndDefault) WithStatusCode(code int) *GetStartEndDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the get start end default response
func (o *GetStartEndDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the get start end default response
func (o *GetStartEndDefault) WithPayload(payload *models.Error) *GetStartEndDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get start end default response
func (o *GetStartEndDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetStartEndDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		if err := producer.Produce(rw, o.Payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

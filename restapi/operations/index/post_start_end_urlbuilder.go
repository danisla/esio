package index

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"errors"
	"net/url"
	"strings"

	"github.com/go-openapi/swag"
)

// PostStartEndURL generates an URL for the post start end operation
type PostStartEndURL struct {
	End   int64
	Start int64

	RepoPattern *string
	Resolution  *string

	// avoid unkeyed usage
	_ struct{}
}

// Build a url path and query string
func (o *PostStartEndURL) Build() (*url.URL, error) {
	var result url.URL

	var _path = "/{start}/{end}"

	end := swag.FormatInt64(o.End)
	if end != "" {
		_path = strings.Replace(_path, "{end}", end, -1)
	} else {
		return nil, errors.New("End is required on PostStartEndURL")
	}
	start := swag.FormatInt64(o.Start)
	if start != "" {
		_path = strings.Replace(_path, "{start}", start, -1)
	} else {
		return nil, errors.New("Start is required on PostStartEndURL")
	}
	result.Path = _path

	qs := make(url.Values)

	var repoPattern string
	if o.RepoPattern != nil {
		repoPattern = *o.RepoPattern
	}
	if repoPattern != "" {
		qs.Set("repo_pattern", repoPattern)
	}

	var resolution string
	if o.Resolution != nil {
		resolution = *o.Resolution
	}
	if resolution != "" {
		qs.Set("resolution", resolution)
	}

	result.RawQuery = qs.Encode()

	return &result, nil
}

// Must is a helper function to panic when the url builder returns an error
func (o *PostStartEndURL) Must(u *url.URL, err error) *url.URL {
	if err != nil {
		panic(err)
	}
	if u == nil {
		panic("url can't be nil")
	}
	return u
}

// String returns the string representation of the path with query string
func (o *PostStartEndURL) String() string {
	return o.Must(o.Build()).String()
}

// BuildFull builds a full url with scheme, host, path and query string
func (o *PostStartEndURL) BuildFull(scheme, host string) (*url.URL, error) {
	if scheme == "" {
		return nil, errors.New("scheme is required for a full url on PostStartEndURL")
	}
	if host == "" {
		return nil, errors.New("host is required for a full url on PostStartEndURL")
	}

	base, err := o.Build()
	if err != nil {
		return nil, err
	}

	base.Scheme = scheme
	base.Host = host
	return base, nil
}

// StringFull returns the string representation of a complete url
func (o *PostStartEndURL) StringFull(scheme, host string) string {
	return o.Must(o.BuildFull(scheme, host)).String()
}

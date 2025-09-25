package caldav

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// CalDAV discovery XML structures

// PropFindRequest represents a PROPFIND request for calendar discovery
type PropFindRequest struct {
	XMLName xml.Name `xml:"DAV: propfind"`
	Prop    DiscoveryProp `xml:"DAV: prop"`
}

type DiscoveryProp struct {
	CurrentUserPrincipal *CurrentUserPrincipal `xml:"DAV: current-user-principal,omitempty"`
	CalendarHomeSet      *CalendarHomeSet      `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set,omitempty"`
	DisplayName          *DisplayName          `xml:"DAV: displayname,omitempty"`
	ResourceType         *ResourceType         `xml:"DAV: resourcetype,omitempty"`
	SupportedCalendarComponentSet *SupportedCalendarComponentSet `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set,omitempty"`
}

type CurrentUserPrincipal struct{}
type CalendarHomeSet struct{}
type DisplayName struct{}
type ResourceType struct{}
type SupportedCalendarComponentSet struct{}

// PropFindResponse structures
type PropFindMultiStatus struct {
	XMLName   xml.Name            `xml:"DAV: multistatus"`
	Responses []PropFindResponse  `xml:"DAV: response"`
}

type PropFindResponse struct {
	Href     string           `xml:"DAV: href"`
	PropStat PropFindPropStat `xml:"DAV: propstat"`
}

type PropFindPropStat struct {
	Prop   PropFindResponseProp `xml:"DAV: prop"`
	Status string               `xml:"DAV: status"`
}

type PropFindResponseProp struct {
	CurrentUserPrincipal      *CurrentUserPrincipalResponse      `xml:"DAV: current-user-principal,omitempty"`
	CalendarHomeSet           *CalendarHomeSetResponse           `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set,omitempty"`
	DisplayName               string                             `xml:"DAV: displayname,omitempty"`
	ResourceType              *ResourceTypeResponse              `xml:"DAV: resourcetype,omitempty"`
	SupportedCalendarComponentSet *SupportedCalendarComponentSetResponse `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set,omitempty"`
}

type CurrentUserPrincipalResponse struct {
	Href string `xml:"DAV: href,omitempty"`
}

type CalendarHomeSetResponse struct {
	Href string `xml:"DAV: href,omitempty"`
}

type ResourceTypeResponse struct {
	Collection string `xml:"DAV: collection,omitempty"`
	Calendar   string `xml:"urn:ietf:params:xml:ns:caldav calendar,omitempty"`
}

type SupportedCalendarComponentSetResponse struct {
	Comp []CalendarComponent `xml:"urn:ietf:params:xml:ns:caldav comp,omitempty"`
}

type CalendarComponent struct {
	Name string `xml:"name,attr"`
}

// CalendarInfo represents discovered calendar information
type CalendarInfo struct {
	URL         string
	DisplayName string
	Components  []string // VEVENT, VTODO, etc.
}

// discoverPrincipalURL finds the current user principal URL
func (c *Client) discoverPrincipalURL() (string, error) {
	propFindXML := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:current-user-principal />
  </D:prop>
</D:propfind>`

	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	req, err := http.NewRequest("PROPFIND", baseURL.String(), strings.NewReader(propFindXML))
	if err != nil {
		return "", fmt.Errorf("failed to create PROPFIND request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "0")

	resp, err := c.executeRequest(req)
	if err != nil {
		return "", fmt.Errorf("PROPFIND request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("PROPFIND request failed with status %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PROPFIND response: %w", err)
	}

	var multiStatus PropFindMultiStatus
	if err := xml.Unmarshal(respBody, &multiStatus); err != nil {
		return "", fmt.Errorf("failed to parse PROPFIND response XML: %w", err)
	}

	for _, response := range multiStatus.Responses {
		if response.PropStat.Prop.CurrentUserPrincipal != nil &&
		   response.PropStat.Prop.CurrentUserPrincipal.Href != "" {
			principalURL := response.PropStat.Prop.CurrentUserPrincipal.Href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(principalURL, "http") {
				principalURL = baseURL.ResolveReference(&url.URL{Path: principalURL}).String()
			}

			return principalURL, nil
		}
	}

	// Fallback: try to use the base URL as principal
	return baseURL.String(), nil
}

// discoverCalendarHomeSet finds the calendar home set URL from principal
func (c *Client) discoverCalendarHomeSet(principalURL string) (string, error) {
	propFindXML := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <C:calendar-home-set />
  </D:prop>
</D:propfind>`

	req, err := http.NewRequest("PROPFIND", principalURL, strings.NewReader(propFindXML))
	if err != nil {
		return "", fmt.Errorf("failed to create PROPFIND request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "0")

	resp, err := c.executeRequest(req)
	if err != nil {
		return "", fmt.Errorf("PROPFIND request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("PROPFIND request failed with status %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PROPFIND response: %w", err)
	}

	var multiStatus PropFindMultiStatus
	if err := xml.Unmarshal(respBody, &multiStatus); err != nil {
		return "", fmt.Errorf("failed to parse PROPFIND response XML: %w", err)
	}

	baseURL, _ := url.Parse(c.baseURL)

	for _, response := range multiStatus.Responses {
		if response.PropStat.Prop.CalendarHomeSet != nil &&
		   response.PropStat.Prop.CalendarHomeSet.Href != "" {
			calendarHomeURL := response.PropStat.Prop.CalendarHomeSet.Href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(calendarHomeURL, "http") {
				calendarHomeURL = baseURL.ResolveReference(&url.URL{Path: calendarHomeURL}).String()
			}

			return calendarHomeURL, nil
		}
	}

	// Fallback: use principal URL if no calendar-home-set found
	return principalURL, nil
}

// discoverCalendarCollections finds all calendar collections in the calendar home set
func (c *Client) discoverCalendarCollections(calendarHomeURL string) ([]CalendarInfo, error) {
	propFindXML := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:displayname />
    <D:resourcetype />
    <C:supported-calendar-component-set />
  </D:prop>
</D:propfind>`

	req, err := http.NewRequest("PROPFIND", calendarHomeURL, strings.NewReader(propFindXML))
	if err != nil {
		return nil, fmt.Errorf("failed to create PROPFIND request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "1")

	resp, err := c.executeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PROPFIND request failed with status %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PROPFIND response: %w", err)
	}

	var multiStatus PropFindMultiStatus
	if err := xml.Unmarshal(respBody, &multiStatus); err != nil {
		return nil, fmt.Errorf("failed to parse PROPFIND response XML: %w", err)
	}

	var calendars []CalendarInfo
	baseURL, _ := url.Parse(c.baseURL)

	for _, response := range multiStatus.Responses {
		// Check if this is a calendar collection
		if response.PropStat.Prop.ResourceType != nil &&
		   response.PropStat.Prop.ResourceType.Calendar != "" {

			calendarURL := response.Href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(calendarURL, "http") {
				calendarURL = baseURL.ResolveReference(&url.URL{Path: calendarURL}).String()
			}

			displayName := response.PropStat.Prop.DisplayName
			if displayName == "" {
				// Extract name from URL path
				if u, err := url.Parse(calendarURL); err == nil {
					displayName = strings.Trim(u.Path, "/")
					if idx := strings.LastIndex(displayName, "/"); idx >= 0 {
						displayName = displayName[idx+1:]
					}
				}
			}

			var components []string
			if response.PropStat.Prop.SupportedCalendarComponentSet != nil {
				for _, comp := range response.PropStat.Prop.SupportedCalendarComponentSet.Comp {
					components = append(components, comp.Name)
				}
			} else {
				// Default to VEVENT and VTODO if not specified
				components = []string{"VEVENT", "VTODO"}
			}

			calendars = append(calendars, CalendarInfo{
				URL:         calendarURL,
				DisplayName: displayName,
				Components:  components,
			})
		}
	}

	return calendars, nil
}

// executeRequest executes an HTTP request with proper authentication
func (c *Client) executeRequest(req *http.Request) (*http.Response, error) {
	if c.httpClient != nil {
		// Use OAuth HTTP client
		return c.httpClient.Do(req)
	} else if c.username != "" && c.password != "" {
		// For basic auth, set auth header
		req.SetBasicAuth(c.username, c.password)
		httpClient := &http.Client{}
		return httpClient.Do(req)
	} else {
		return nil, fmt.Errorf("no HTTP client configured")
	}
}

// DiscoverCalendars performs the full CalDAV discovery process
func (c *Client) DiscoverCalendars() ([]CalendarInfo, error) {
	// Step 1: Discover principal URL
	principalURL, err := c.discoverPrincipalURL()
	if err != nil {
		return nil, fmt.Errorf("failed to discover principal URL: %w", err)
	}

	// Step 2: Discover calendar home set
	calendarHomeURL, err := c.discoverCalendarHomeSet(principalURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendar home set: %w", err)
	}

	// Step 3: Discover calendar collections
	calendars, err := c.discoverCalendarCollections(calendarHomeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover calendars: %w", err)
	}

	return calendars, nil
}
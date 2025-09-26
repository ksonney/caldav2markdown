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
	Href      string             `xml:"DAV: href"`
	PropStats []PropFindPropStat `xml:"DAV: propstat"`
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

// parseDiscoveryMultiStatus safely extracts data from discovery multistatus responses
func parseDiscoveryMultiStatus(multiStatus *PropFindMultiStatus, traceWebCalls bool) map[string]PropFindResponseProp {
	results := make(map[string]PropFindResponseProp)
	var errors []string

	if traceWebCalls {
		fmt.Printf("=== Parsing Discovery MultiStatus Response ===\n")
		fmt.Printf("Total responses: %d\n", len(multiStatus.Responses))
	}

	for i, response := range multiStatus.Responses {
		if traceWebCalls {
			fmt.Printf("Response %d: %s\n", i+1, response.Href)
			fmt.Printf("  PropStats count: %d\n", len(response.PropStats))
		}

		// Find the first successful propstat with the data we need
		for j, propStat := range response.PropStats {
			if traceWebCalls {
				fmt.Printf("  PropStat %d: Status = %s\n", j+1, propStat.Status)
			}

			// Check propstat status - should be "HTTP/1.1 200 OK" for successful responses
			if !strings.Contains(propStat.Status, "200") {
				errors = append(errors, fmt.Sprintf("Resource %s returned status: %s", response.Href, propStat.Status))
				continue
			}

			// Store the successful prop data
			results[response.Href] = propStat.Prop
			if traceWebCalls {
				fmt.Printf("    Successfully parsed discovery data\n")
			}
			break // Only need one successful propstat per response
		}
	}

	if traceWebCalls {
		fmt.Printf("=== Discovery MultiStatus Parsing Complete ===\n")
		fmt.Printf("Success count: %d\n", len(results))
		fmt.Printf("Error count: %d\n", len(errors))
	}

	// Log any errors but don't fail the entire operation
	if len(errors) > 0 {
		fmt.Printf("Warning: Some discovery resources had errors:\n")
		for _, errMsg := range errors {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	return results
}

// isCalendarCollection determines if a resource is a calendar collection
// by checking both the resource type and supported calendar components
func isCalendarCollection(prop PropFindResponseProp, traceWebCalls bool) (bool, string) {
	// Check 1: Look for calendar components first (most reliable indicator)
	hasCalendarComponents := false
	if prop.SupportedCalendarComponentSet != nil && len(prop.SupportedCalendarComponentSet.Comp) > 0 {
		// Check if any of the supported components are calendar-related
		for _, comp := range prop.SupportedCalendarComponentSet.Comp {
			switch comp.Name {
			case "VEVENT", "VTODO", "VJOURNAL", "VFREEBUSY":
				hasCalendarComponents = true
				break
			}
		}
	}

	// Check 2: Look for explicit calendar resource type
	hasCalendarResourceType := false
	if prop.ResourceType != nil {
		// In CalDAV, a calendar collection should have both <collection/> and <calendar/> resource types
		// We need to be more careful here - not every collection is a calendar

		// For now, we'll be conservative and only consider explicit calendar resource types
		// This means we rely more heavily on supported components for detection
		// In the future, we could add more heuristics here if needed

		// If we explicitly see calendar resource type indicators, consider it
		// (This is a placeholder for future enhancement - currently always false)
		hasCalendarResourceType = false // Be conservative - rely on components primarily
	}

	// Primary check: calendar components (most reliable)
	if hasCalendarComponents {
		if traceWebCalls {
			fmt.Printf("    Calendar detected via supported components: %v\n",
				getComponentNames(prop.SupportedCalendarComponentSet.Comp))
		}
		return true, "supported-components"
	}

	// Secondary check: resource type if no components available
	if hasCalendarResourceType && prop.ResourceType != nil {
		// Only consider it a calendar via resource type if it seems like a collection
		// This prevents false positives on non-calendar resources
		if traceWebCalls {
			fmt.Printf("    Calendar detected via resource type (collection: %t)\n",
				prop.ResourceType.Collection == "")
		}
		return true, "resource-type"
	}

	return false, ""
}

// getComponentNames extracts component names for logging
func getComponentNames(components []CalendarComponent) []string {
	names := make([]string, len(components))
	for i, comp := range components {
		names[i] = comp.Name
	}
	return names
}

// isValidCalendarURL filters out URLs that are not proper CalDAV calendar collections
func isValidCalendarURL(url string) bool {
	// Convert URL to lowercase for case-insensitive comparison
	lowerURL := strings.ToLower(url)

	// Filter out ICS endpoints (direct calendar file downloads)
	if strings.HasSuffix(lowerURL, ".ics") {
		return false
	}

	// Filter out XML endpoints (not calendar collections)
	if strings.HasSuffix(lowerURL, ".xml") {
		return false
	}

	// Filter out other common non-calendar file extensions
	invalidSuffixes := []string{
		".json", ".csv", ".txt", ".html", ".htm",
		".php", ".asp", ".jsp", ".cgi",
	}

	for _, suffix := range invalidSuffixes {
		if strings.HasSuffix(lowerURL, suffix) {
			return false
		}
	}

	// Filter out URLs that look like file downloads rather than collections
	// These often contain query parameters for export
	if strings.Contains(lowerURL, "export") && (strings.Contains(lowerURL, "?") || strings.Contains(lowerURL, "&")) {
		return false
	}

	// URLs ending with obvious file download patterns
	if strings.Contains(lowerURL, "download") || strings.Contains(lowerURL, "attachment") {
		return false
	}

	return true
}

// normalizeCalendarURL normalizes URLs for de-duplication by removing trailing slashes
// and converting to a canonical form
func normalizeCalendarURL(url string) string {
	// Remove trailing slash for consistent comparison
	normalized := strings.TrimRight(url, "/")

	// Convert to lowercase for case-insensitive comparison
	normalized = strings.ToLower(normalized)

	return normalized
}

// deduplicateCalendars removes duplicate calendars based on normalized URLs
func deduplicateCalendars(calendars []CalendarInfo, traceWebCalls bool) []CalendarInfo {
	if len(calendars) <= 1 {
		return calendars
	}

	seen := make(map[string]bool)
	var deduplicated []CalendarInfo
	duplicatesFound := 0

	for _, calendar := range calendars {
		// Filter out invalid URLs first
		if !isValidCalendarURL(calendar.URL) {
			if traceWebCalls {
				fmt.Printf("  Filtered out invalid calendar URL: %s\n", calendar.URL)
			}
			continue
		}

		// Normalize URL for comparison
		normalizedURL := normalizeCalendarURL(calendar.URL)

		if seen[normalizedURL] {
			duplicatesFound++
			if traceWebCalls {
				fmt.Printf("  Duplicate calendar found: %s (normalized: %s)\n", calendar.URL, normalizedURL)
			}
			continue
		}

		seen[normalizedURL] = true
		deduplicated = append(deduplicated, calendar)

		if traceWebCalls {
			fmt.Printf("  Keeping calendar: %s (%s)\n", calendar.DisplayName, calendar.URL)
		}
	}

	if traceWebCalls && duplicatesFound > 0 {
		fmt.Printf("Removed %d duplicate/invalid calendars, keeping %d unique calendars\n",
			duplicatesFound, len(deduplicated))
	}

	return deduplicated
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

	// Parse multistatus response with enhanced error handling
	results := parseDiscoveryMultiStatus(&multiStatus, c.traceWebCalls)

	for href, prop := range results {
		if prop.CurrentUserPrincipal != nil &&
		   prop.CurrentUserPrincipal.Href != "" {
			principalURL := prop.CurrentUserPrincipal.Href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(principalURL, "http") {
				principalURL = baseURL.ResolveReference(&url.URL{Path: principalURL}).String()
			}

			if c.traceWebCalls {
				fmt.Printf("Found principal URL: %s from resource: %s\n", principalURL, href)
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

	// Parse multistatus response with enhanced error handling
	results := parseDiscoveryMultiStatus(&multiStatus, c.traceWebCalls)

	for href, prop := range results {
		if prop.CalendarHomeSet != nil &&
		   prop.CalendarHomeSet.Href != "" {
			calendarHomeURL := prop.CalendarHomeSet.Href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(calendarHomeURL, "http") {
				calendarHomeURL = baseURL.ResolveReference(&url.URL{Path: calendarHomeURL}).String()
			}

			if c.traceWebCalls {
				fmt.Printf("Found calendar home set: %s from resource: %s\n", calendarHomeURL, href)
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

	// Parse multistatus response with enhanced error handling
	results := parseDiscoveryMultiStatus(&multiStatus, c.traceWebCalls)

	if c.traceWebCalls {
		fmt.Printf("Discovered %d calendar collection candidates\n", len(results))
	}

	for href, prop := range results {
		if c.traceWebCalls {
			fmt.Printf("  Evaluating resource: %s\n", href)
		}

		// Check if this is a calendar collection using enhanced detection
		isCalendar, detectionMethod := isCalendarCollection(prop, c.traceWebCalls)
		if isCalendar {
			calendarURL := href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(calendarURL, "http") {
				calendarURL = baseURL.ResolveReference(&url.URL{Path: calendarURL}).String()
			}

			displayName := prop.DisplayName
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
			if prop.SupportedCalendarComponentSet != nil {
				for _, comp := range prop.SupportedCalendarComponentSet.Comp {
					components = append(components, comp.Name)
				}
			} else {
				// Default to VEVENT and VTODO if not specified
				components = []string{"VEVENT", "VTODO"}
			}

			calendar := CalendarInfo{
				URL:         calendarURL,
				DisplayName: displayName,
				Components:  components,
			}

			calendars = append(calendars, calendar)

			if c.traceWebCalls {
				fmt.Printf("Found calendar: %s (%s) - Detection: %s - Components: %v\n",
					calendar.DisplayName, calendar.URL, detectionMethod, calendar.Components)
			}
		} else if c.traceWebCalls {
			fmt.Printf("    Not a calendar collection\n")
		}
	}

	if c.traceWebCalls {
		fmt.Printf("Discovered %d calendar collections before de-duplication\n", len(calendars))
	}

	// Apply de-duplication and filtering
	calendars = deduplicateCalendars(calendars, c.traceWebCalls)

	if c.traceWebCalls {
		fmt.Printf("Successfully discovered %d unique calendar collections after filtering\n", len(calendars))
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
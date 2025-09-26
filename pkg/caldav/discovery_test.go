package caldav

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"
)

func TestParseDiscoveryMultiStatus(t *testing.T) {
	// Test case 1: Successful principal discovery
	principalXML := `<d:multistatus xmlns:d="DAV:">
		<d:response>
			<d:href>/</d:href>
			<d:propstat>
				<d:prop>
					<d:current-user-principal>
						<d:href>/principals/users/testuser/</d:href>
					</d:current-user-principal>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var multiStatus PropFindMultiStatus
	err := xml.Unmarshal([]byte(principalXML), &multiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal principal XML: %v", err)
	}

	results := parseDiscoveryMultiStatus(&multiStatus, false)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results["/"].CurrentUserPrincipal == nil ||
	   results["/"].CurrentUserPrincipal.Href != "/principals/users/testuser/" {
		t.Error("Failed to parse current-user-principal correctly")
	}

	// Test case 2: Calendar home set discovery
	homeSetXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/principals/users/testuser/</d:href>
			<d:propstat>
				<d:prop>
					<cal:calendar-home-set>
						<d:href>/calendars/testuser/</d:href>
					</cal:calendar-home-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var homeSetMultiStatus PropFindMultiStatus
	err = xml.Unmarshal([]byte(homeSetXML), &homeSetMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal home set XML: %v", err)
	}

	results = parseDiscoveryMultiStatus(&homeSetMultiStatus, false)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results["/principals/users/testuser/"].CalendarHomeSet == nil ||
	   results["/principals/users/testuser/"].CalendarHomeSet.Href != "/calendars/testuser/" {
		t.Error("Failed to parse calendar-home-set correctly")
	}

	// Test case 3: Calendar collection discovery with multiple calendars
	calendarCollectionsXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendars/testuser/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Calendar Home</d:displayname>
					<d:resourcetype>
						<d:collection/>
					</d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
						<cal:comp name="VTODO"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/work/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Work Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var collectionsMultiStatus PropFindMultiStatus
	err = xml.Unmarshal([]byte(calendarCollectionsXML), &collectionsMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal collections XML: %v", err)
	}

	results = parseDiscoveryMultiStatus(&collectionsMultiStatus, false)
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check personal calendar
	personalProp := results["/calendars/testuser/personal/"]
	if personalProp.DisplayName != "Personal Calendar" {
		t.Errorf("Expected 'Personal Calendar', got '%s'", personalProp.DisplayName)
	}
	if personalProp.ResourceType == nil {
		t.Error("Personal calendar should have resource type")
	} else {
		// For calendar resource type, we just need to check that the Calendar field exists
		// The XML unmarshaling will populate it even if it's empty in the XML
		t.Logf("ResourceType found: Collection='%s', Calendar='%s'",
			personalProp.ResourceType.Collection, personalProp.ResourceType.Calendar)
	}
	if len(personalProp.SupportedCalendarComponentSet.Comp) != 2 {
		t.Errorf("Expected 2 components for personal calendar, got %d",
			len(personalProp.SupportedCalendarComponentSet.Comp))
	}

	// Check work calendar
	workProp := results["/calendars/testuser/work/"]
	if workProp.DisplayName != "Work Calendar" {
		t.Errorf("Expected 'Work Calendar', got '%s'", workProp.DisplayName)
	}
	if len(workProp.SupportedCalendarComponentSet.Comp) != 1 {
		t.Errorf("Expected 1 component for work calendar, got %d",
			len(workProp.SupportedCalendarComponentSet.Comp))
	}

	// Test case 4: Mixed response with some errors
	mixedXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendars/testuser/personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/private/</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 403 Forbidden</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/deleted/</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 404 Not Found</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var mixedMultiStatus PropFindMultiStatus
	err = xml.Unmarshal([]byte(mixedXML), &mixedMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal mixed XML: %v", err)
	}

	results = parseDiscoveryMultiStatus(&mixedMultiStatus, false)
	if len(results) != 1 {
		t.Errorf("Expected 1 successful result from mixed response, got %d", len(results))
	}

	if results["/calendars/testuser/personal/"].DisplayName != "Personal Calendar" {
		t.Error("Failed to get successful result from mixed response")
	}

	// Test case 5: Multiple propstat elements in single response
	multiPropstatXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendars/testuser/personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:getetag>"12345"</d:getetag>
				</d:prop>
				<d:status>HTTP/1.1 404 Not Found</d:status>
			</d:propstat>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var multiPropstatMultiStatus PropFindMultiStatus
	err = xml.Unmarshal([]byte(multiPropstatXML), &multiPropstatMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal multi-propstat XML: %v", err)
	}

	results = parseDiscoveryMultiStatus(&multiPropstatMultiStatus, false)
	if len(results) != 1 {
		t.Errorf("Expected 1 result from multi-propstat response, got %d", len(results))
	}

	if results["/calendars/testuser/personal/"].DisplayName != "Personal Calendar" {
		t.Error("Failed to get correct propstat from multi-propstat response")
	}

	// Test case 6: All error responses
	errorXML := `<d:multistatus xmlns:d="DAV:">
		<d:response>
			<d:href>/calendars/testuser/private/</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 403 Forbidden</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/deleted/</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 404 Not Found</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var errorMultiStatus PropFindMultiStatus
	err = xml.Unmarshal([]byte(errorXML), &errorMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal error XML: %v", err)
	}

	results = parseDiscoveryMultiStatus(&errorMultiStatus, false)
	if len(results) != 0 {
		t.Errorf("Expected 0 results from all-error response, got %d", len(results))
	}
}

func TestIsCalendarCollection(t *testing.T) {
	// Test case 1: Calendar with both resource type and supported components
	// (components should take priority as they're more reliable)
	prop1 := PropFindResponseProp{
		ResourceType: &ResourceTypeResponse{
			Collection: "",
			Calendar:   "",
		},
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VEVENT"},
				{Name: "VTODO"},
			},
		},
	}

	isCalendar, method := isCalendarCollection(prop1, false)
	if !isCalendar {
		t.Error("Expected calendar with components and resource type to be detected")
	}
	if method != "supported-components" {
		t.Errorf("Expected detection method 'supported-components', got '%s'", method)
	}

	// Test case 2: Calendar detected via supported components only (missing resource type)
	prop2 := PropFindResponseProp{
		ResourceType: nil,
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VEVENT"},
				{Name: "VTODO"},
			},
		},
	}

	isCalendar, method = isCalendarCollection(prop2, false)
	if !isCalendar {
		t.Error("Expected calendar with only supported components to be detected")
	}
	if method != "supported-components" {
		t.Errorf("Expected detection method 'supported-components', got '%s'", method)
	}

	// Test case 3: Collection with calendar components (resource type present)
	prop3 := PropFindResponseProp{
		ResourceType: &ResourceTypeResponse{
			Collection: "",
			Calendar:   "", // Empty but present
		},
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VEVENT"},
			},
		},
	}

	isCalendar, method = isCalendarCollection(prop3, false)
	if !isCalendar {
		t.Error("Expected collection with calendar components to be detected")
	}
	// Should detect via supported-components since they take priority
	if method != "supported-components" {
		t.Errorf("Expected detection method 'supported-components', got '%s'", method)
	}

	// Test case 4: Collection with VJOURNAL component
	prop4 := PropFindResponseProp{
		ResourceType: &ResourceTypeResponse{
			Collection: "",
			Calendar:   "", // Empty
		},
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VJOURNAL"},
			},
		},
	}

	isCalendar, method = isCalendarCollection(prop4, false)
	if !isCalendar {
		t.Error("Expected calendar with VJOURNAL component to be detected")
	}

	// Test case 5: Collection with non-calendar components
	prop5 := PropFindResponseProp{
		ResourceType: nil,
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VCARD"}, // Non-calendar component
				{Name: "UNKNOWN"},
			},
		},
	}

	isCalendar, method = isCalendarCollection(prop5, false)
	if isCalendar {
		t.Error("Expected non-calendar collection to not be detected as calendar")
	}
	if method != "" {
		t.Errorf("Expected empty detection method for non-calendar, got '%s'", method)
	}

	// Test case 6: Collection with only resource type (no supported components)
	// With conservative logic, this should NOT be detected as a calendar
	prop6 := PropFindResponseProp{
		ResourceType: &ResourceTypeResponse{
			Collection: "",
			Calendar:   "", // Empty but present
		},
		SupportedCalendarComponentSet: nil,
	}

	isCalendar, method = isCalendarCollection(prop6, false)
	if isCalendar {
		t.Error("Expected plain collection without components to NOT be detected as calendar")
	}
	if method != "" {
		t.Errorf("Expected empty detection method for plain collection, got '%s'", method)
	}

	// Test case 7: Non-collection resource with calendar components
	prop7 := PropFindResponseProp{
		ResourceType: nil,
		SupportedCalendarComponentSet: &SupportedCalendarComponentSetResponse{
			Comp: []CalendarComponent{
				{Name: "VEVENT"},
			},
		},
	}

	isCalendar, method = isCalendarCollection(prop7, false)
	if !isCalendar {
		t.Error("Expected resource with calendar components to be detected")
	}
	if method != "supported-components" {
		t.Errorf("Expected detection method 'supported-components', got '%s'", method)
	}

	// Test case 8: Empty everything
	prop8 := PropFindResponseProp{
		ResourceType:                  nil,
		SupportedCalendarComponentSet: nil,
	}

	isCalendar, method = isCalendarCollection(prop8, false)
	if isCalendar {
		t.Error("Expected empty resource to not be detected as calendar")
	}
	if method != "" {
		t.Errorf("Expected empty detection method for empty resource, got '%s'", method)
	}
}

func TestGetComponentNames(t *testing.T) {
	components := []CalendarComponent{
		{Name: "VEVENT"},
		{Name: "VTODO"},
		{Name: "VJOURNAL"},
	}

	names := getComponentNames(components)
	expected := []string{"VEVENT", "VTODO", "VJOURNAL"}

	if len(names) != len(expected) {
		t.Errorf("Expected %d component names, got %d", len(expected), len(names))
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected component name '%s' at index %d, got '%s'", expected[i], i, name)
		}
	}
}

func TestDiscoverCalendarCollectionsWithFallback(t *testing.T) {
	// Test the full discovery process with calendars that have missing resource types
	// but have supported components
	calendarCollectionsXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendars/testuser/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Calendar Home</d:displayname>
					<d:resourcetype>
						<d:collection/>
					</d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
						<cal:comp name="VTODO"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/legacy/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Legacy Calendar</d:displayname>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/contacts/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Address Book</d:displayname>
					<d:resourcetype>
						<d:collection/>
					</d:resourcetype>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var collectionsMultiStatus PropFindMultiStatus
	err := xml.Unmarshal([]byte(calendarCollectionsXML), &collectionsMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal collections XML: %v", err)
	}

	results := parseDiscoveryMultiStatus(&collectionsMultiStatus, false)
	if len(results) != 4 {
		t.Errorf("Expected 4 discovery results, got %d", len(results))
	}

	// Count how many would be detected as calendars
	calendarsFound := 0
	for href, prop := range results {
		isCalendar, method := isCalendarCollection(prop, true)
		if isCalendar {
			calendarsFound++
			t.Logf("Found calendar: %s via %s", href, method)
		}
	}

	// We expect 2 calendars:
	// 1. /calendars/testuser/personal/ - has both resource type and components
	// 2. /calendars/testuser/legacy/ - has only components (should be detected)
	// We should NOT detect:
	// - /calendars/testuser/ - has resource type but no calendar components (conservative approach)
	// - /calendars/testuser/contacts/ - is just a collection with no calendar indicators

	expectedCalendars := 2
	if calendarsFound != expectedCalendars {
		t.Errorf("Expected %d calendars to be detected, got %d", expectedCalendars, calendarsFound)
	}

	// Verify the legacy calendar (with only components) is detected
	legacyProp := results["/calendars/testuser/legacy/"]
	isLegacyCalendar, legacyMethod := isCalendarCollection(legacyProp, false)
	if !isLegacyCalendar {
		t.Error("Expected legacy calendar with only components to be detected")
	}
	if legacyMethod != "supported-components" {
		t.Errorf("Expected legacy calendar detection via 'supported-components', got '%s'", legacyMethod)
	}

	// Verify the contacts collection is NOT detected as a calendar
	contactsProp := results["/calendars/testuser/contacts/"]
	isContactsCalendar, _ := isCalendarCollection(contactsProp, false)
	if isContactsCalendar {
		t.Error("Expected contacts collection to NOT be detected as calendar")
	}
}

func TestIsValidCalendarURL(t *testing.T) {
	// Test valid calendar URLs
	validURLs := []string{
		"https://server.com/calendars/user/personal/",
		"https://server.com/caldav/calendars/work",
		"https://example.org/cal/home/",
		"HTTP://EXAMPLE.COM/CALENDARS/USER/",  // case insensitive
	}

	for _, url := range validURLs {
		if !isValidCalendarURL(url) {
			t.Errorf("Expected URL to be valid: %s", url)
		}
	}

	// Test invalid calendar URLs
	invalidURLs := []string{
		"https://server.com/calendars/calendar.ics",     // ICS file
		"https://server.com/calendars/calendar.xml",     // XML file
		"https://server.com/calendars/calendar.json",    // JSON file
		"https://server.com/calendars/export.csv",       // CSV file
		"https://server.com/calendars/calendar.txt",     // TXT file
		"https://server.com/calendars/calendar.html",    // HTML file
		"https://server.com/calendars/calendar.htm",     // HTM file
		"https://server.com/calendars/script.php",       // PHP file
		"https://server.com/calendars/page.asp",         // ASP file
		"https://server.com/calendars/app.jsp",          // JSP file
		"https://server.com/calendars/script.cgi",       // CGI file
		"https://server.com/export?format=ics&cal=1",    // Export with query
		"https://server.com/calendar/download/cal.ics",  // Download pattern
		"https://server.com/attachment/calendar.ics",    // Attachment pattern
		"https://SERVER.COM/CALENDARS/CALENDAR.ICS",     // Case insensitive
	}

	for _, url := range invalidURLs {
		if isValidCalendarURL(url) {
			t.Errorf("Expected URL to be invalid: %s", url)
		}
	}
}

func TestNormalizeCalendarURL(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"https://example.com/calendars/user/", "https://example.com/calendars/user"},
		{"https://example.com/calendars/user", "https://example.com/calendars/user"},
		{"HTTPS://EXAMPLE.COM/CALENDARS/USER/", "https://example.com/calendars/user"},
		{"https://Example.Com/Calendars/User/", "https://example.com/calendars/user"},
		{"https://example.com/calendars/user///", "https://example.com/calendars/user"},
	}

	for _, tc := range testCases {
		result := normalizeCalendarURL(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeCalendarURL(%s) = %s; expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestDeduplicateCalendars(t *testing.T) {
	// Test case 1: No duplicates
	calendars1 := []CalendarInfo{
		{URL: "https://example.com/cal1/", DisplayName: "Calendar 1", Components: []string{"VEVENT"}},
		{URL: "https://example.com/cal2/", DisplayName: "Calendar 2", Components: []string{"VTODO"}},
	}

	result1 := deduplicateCalendars(calendars1, false)
	if len(result1) != 2 {
		t.Errorf("Expected 2 calendars, got %d", len(result1))
	}

	// Test case 2: Exact duplicates
	calendars2 := []CalendarInfo{
		{URL: "https://example.com/cal1/", DisplayName: "Calendar 1", Components: []string{"VEVENT"}},
		{URL: "https://example.com/cal1/", DisplayName: "Calendar 1 Dup", Components: []string{"VEVENT"}},
		{URL: "https://example.com/cal2/", DisplayName: "Calendar 2", Components: []string{"VTODO"}},
	}

	result2 := deduplicateCalendars(calendars2, false)
	if len(result2) != 2 {
		t.Errorf("Expected 2 calendars after dedup, got %d", len(result2))
	}

	// Test case 3: Case-insensitive duplicates and trailing slash differences
	calendars3 := []CalendarInfo{
		{URL: "https://example.com/cal1/", DisplayName: "Calendar 1", Components: []string{"VEVENT"}},
		{URL: "HTTPS://EXAMPLE.COM/CAL1", DisplayName: "Calendar 1 Case", Components: []string{"VEVENT"}},
		{URL: "https://Example.Com/Cal1/", DisplayName: "Calendar 1 Mixed", Components: []string{"VEVENT"}},
		{URL: "https://example.com/cal2/", DisplayName: "Calendar 2", Components: []string{"VTODO"}},
	}

	result3 := deduplicateCalendars(calendars3, false)
	if len(result3) != 2 {
		t.Errorf("Expected 2 calendars after case-insensitive dedup, got %d", len(result3))
	}

	// Test case 4: Invalid URLs filtered out
	calendars4 := []CalendarInfo{
		{URL: "https://example.com/cal1/", DisplayName: "Valid Calendar", Components: []string{"VEVENT"}},
		{URL: "https://example.com/calendar.ics", DisplayName: "ICS File", Components: []string{"VEVENT"}},
		{URL: "https://example.com/calendar.xml", DisplayName: "XML File", Components: []string{"VEVENT"}},
		{URL: "https://example.com/export?format=ics", DisplayName: "Export URL", Components: []string{"VEVENT"}},
	}

	result4 := deduplicateCalendars(calendars4, false)
	if len(result4) != 1 {
		t.Errorf("Expected 1 calendar after filtering invalid URLs, got %d", len(result4))
	}
	if result4[0].DisplayName != "Valid Calendar" {
		t.Errorf("Expected 'Valid Calendar' to remain, got '%s'", result4[0].DisplayName)
	}

	// Test case 5: Empty list
	calendars5 := []CalendarInfo{}
	result5 := deduplicateCalendars(calendars5, false)
	if len(result5) != 0 {
		t.Errorf("Expected 0 calendars for empty input, got %d", len(result5))
	}

	// Test case 6: Single calendar
	calendars6 := []CalendarInfo{
		{URL: "https://example.com/cal1/", DisplayName: "Single Calendar", Components: []string{"VEVENT"}},
	}
	result6 := deduplicateCalendars(calendars6, false)
	if len(result6) != 1 {
		t.Errorf("Expected 1 calendar for single input, got %d", len(result6))
	}
}

func TestDiscoveryWithDeduplicationAndFiltering(t *testing.T) {
	// Test the full discovery process with duplicates and invalid URLs
	calendarCollectionsXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendars/testuser/personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
						<cal:comp name="VTODO"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/Personal/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Personal Calendar Duplicate</d:displayname>
					<d:resourcetype>
						<d:collection/>
						<cal:calendar/>
					</d:resourcetype>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/work/</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Work Calendar</d:displayname>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/calendar.ics</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>ICS Export</d:displayname>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendars/testuser/export.xml</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>XML Export</d:displayname>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/export?format=ics&amp;cal=personal</d:href>
			<d:propstat>
				<d:prop>
					<d:displayname>Export URL</d:displayname>
					<cal:supported-calendar-component-set>
						<cal:comp name="VEVENT"/>
					</cal:supported-calendar-component-set>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var collectionsMultiStatus PropFindMultiStatus
	err := xml.Unmarshal([]byte(calendarCollectionsXML), &collectionsMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal collections XML: %v", err)
	}

	results := parseDiscoveryMultiStatus(&collectionsMultiStatus, false)
	if len(results) != 6 {
		t.Errorf("Expected 6 discovery results, got %d", len(results))
	}

	// Build calendar list as the discovery process would
	var calendars []CalendarInfo
	baseURL, _ := url.Parse("https://server.example.com")

	for href, prop := range results {
		isCalendar, _ := isCalendarCollection(prop, false)
		if isCalendar {
			calendarURL := href

			// Make absolute URL if it's relative
			if !strings.HasPrefix(calendarURL, "http") {
				calendarURL = baseURL.ResolveReference(&url.URL{Path: calendarURL}).String()
			}

			displayName := prop.DisplayName
			if displayName == "" {
				displayName = href
			}

			var components []string
			if prop.SupportedCalendarComponentSet != nil {
				for _, comp := range prop.SupportedCalendarComponentSet.Comp {
					components = append(components, comp.Name)
				}
			} else {
				components = []string{"VEVENT", "VTODO"}
			}

			calendars = append(calendars, CalendarInfo{
				URL:         calendarURL,
				DisplayName: displayName,
				Components:  components,
			})
		}
	}

	// Should have 6 calendars before de-duplication/filtering
	if len(calendars) != 6 {
		t.Errorf("Expected 6 calendars before filtering, got %d", len(calendars))
	}

	// Apply de-duplication and filtering
	calendars = deduplicateCalendars(calendars, true)

	// Should have 2 calendars after filtering:
	// 1. /calendars/testuser/personal/ (first duplicate wins)
	// 2. /calendars/testuser/work/
	// Filtered out:
	// - /calendars/testuser/Personal/ (duplicate of personal, case-insensitive)
	// - /calendars/testuser/calendar.ics (ICS file)
	// - /calendars/testuser/export.xml (XML file)
	// - /export?format=ics&cal=personal (export URL)

	expectedCalendars := 2
	if len(calendars) != expectedCalendars {
		t.Errorf("Expected %d calendars after filtering, got %d", expectedCalendars, len(calendars))
		for i, cal := range calendars {
			t.Logf("Calendar %d: %s (%s)", i+1, cal.DisplayName, cal.URL)
		}
	}

	// Verify the remaining calendars
	calendarNames := make(map[string]bool)
	for _, cal := range calendars {
		calendarNames[cal.DisplayName] = true
	}

	if !calendarNames["Personal Calendar"] {
		t.Error("Expected 'Personal Calendar' to be in the final list")
	}
	if !calendarNames["Work Calendar"] {
		t.Error("Expected 'Work Calendar' to be in the final list")
	}
}
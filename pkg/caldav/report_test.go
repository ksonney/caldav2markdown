package caldav

import (
	"encoding/xml"
	"testing"
)

func TestParseMultiStatusResponse(t *testing.T) {
	// Test case 1: Successful response with calendar data
	successXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendar/event1.ics</d:href>
			<d:propstat>
				<d:prop>
					<cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:123@example.com
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR</cal:calendar-data>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var multiStatus MultiStatus
	err := xml.Unmarshal([]byte(successXML), &multiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal success XML: %v", err)
	}

	calendarData, err := parseMultiStatusResponse(&multiStatus, false)
	if err != nil {
		t.Fatalf("parseMultiStatusResponse failed: %v", err)
	}

	if len(calendarData) != 1 {
		t.Errorf("Expected 1 calendar data, got %d", len(calendarData))
	}

	// Test case 2: Mixed response with some errors
	mixedXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendar/event1.ics</d:href>
			<d:propstat>
				<d:prop>
					<cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:123@example.com
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR</cal:calendar-data>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendar/event2.ics</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 404 Not Found</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var mixedMultiStatus MultiStatus
	err = xml.Unmarshal([]byte(mixedXML), &mixedMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal mixed XML: %v", err)
	}

	calendarData, err = parseMultiStatusResponse(&mixedMultiStatus, false)
	if err != nil {
		t.Fatalf("parseMultiStatusResponse failed for mixed response: %v", err)
	}

	if len(calendarData) != 1 {
		t.Errorf("Expected 1 calendar data from mixed response, got %d", len(calendarData))
	}

	// Test case 3: All error responses
	errorXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendar/event1.ics</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 404 Not Found</d:status>
			</d:propstat>
		</d:response>
		<d:response>
			<d:href>/calendar/event2.ics</d:href>
			<d:propstat>
				<d:prop>
				</d:prop>
				<d:status>HTTP/1.1 403 Forbidden</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var errorMultiStatus MultiStatus
	err = xml.Unmarshal([]byte(errorXML), &errorMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal error XML: %v", err)
	}

	calendarData, err = parseMultiStatusResponse(&errorMultiStatus, false)
	if err == nil {
		t.Error("Expected error for all-error response, got nil")
	}

	if len(calendarData) != 0 {
		t.Errorf("Expected 0 calendar data from error response, got %d", len(calendarData))
	}

	// Test case 4: Multiple propstat elements in single response
	multiPropstatXML := `<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
		<d:response>
			<d:href>/calendar/event1.ics</d:href>
			<d:propstat>
				<d:prop>
					<d:getetag>"12345"</d:getetag>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
			<d:propstat>
				<d:prop>
					<cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:123@example.com
SUMMARY:Test Event
END:VEVENT
END:VCALENDAR</cal:calendar-data>
				</d:prop>
				<d:status>HTTP/1.1 200 OK</d:status>
			</d:propstat>
		</d:response>
	</d:multistatus>`

	var multiPropstatMultiStatus MultiStatus
	err = xml.Unmarshal([]byte(multiPropstatXML), &multiPropstatMultiStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal multi-propstat XML: %v", err)
	}

	calendarData, err = parseMultiStatusResponse(&multiPropstatMultiStatus, false)
	if err != nil {
		t.Fatalf("parseMultiStatusResponse failed for multi-propstat response: %v", err)
	}

	if len(calendarData) != 1 {
		t.Errorf("Expected 1 calendar data from multi-propstat response, got %d", len(calendarData))
	}
}
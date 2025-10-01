# Example YAML Configurations

This directory contains example YAML configuration files for different use cases with CalDAV2Markdown. These examples demonstrate various authentication methods, multi-source setups, and configuration options.

## Quick Start

```bash
# Copy an example config that matches your needs
cp example-personal.yaml my-config.yaml

# Edit the configuration with your actual credentials
vim my-config.yaml

# Test the configuration
bin/caldav2markdown -config my-config.yaml -test

# Run the conversion
bin/caldav2markdown -config my-config.yaml
```

## Example Configurations

### 📋 `example-multi-source.yaml`
**Comprehensive multi-source demonstration**
- 10 different calendar sources
- All authentication methods (OAuth, basic auth, bearer tokens, custom headers)
- Both CalDAV and ICS sources
- Local and remote sources
- Calendar discovery and filtering
- Proxy configuration examples

**Best for:** Understanding all available features and options

### 👤 `example-personal.yaml`
**Personal use configuration**
- Google Calendar with OAuth
- iCloud Calendar
- Local task files
- Public subscription calendars
- Personal preferences for display options

**Best for:** Individual users with multiple personal calendars

### ✅ `example-task-focused.yaml`
**Task management and project tracking**
- Google Tasks integration
- Local todo files
- Jira/GitHub/Asana integration examples
- Obsidian tasks preset enabled
- Optimized for task tracking workflows

**Best for:** Project managers and task-focused workflows

### 🏢 `example-enterprise.yaml`
**Enterprise and corporate environments**
- Multiple Office 365 tenants
- Exchange on-premises
- Google Workspace
- Salesforce integration
- ServiceNow maintenance calendars
- Corporate proxy configurations
- Security and compliance considerations

**Best for:** Large organizations with multiple calendar systems

### 🔧 `example-development.yaml`
**Development and testing scenarios**
- Multiple environments (prod/staging/dev)
- Test data files
- Mock API endpoints
- Performance testing
- Security testing scenarios
- Error condition testing
- Boundary testing

**Best for:** Developers and QA teams

## Authentication Methods Covered

### CalDAV Sources
- **OAuth 2.0**: Google Calendar, Google Workspace
- **Basic Authentication**: Username/password for most CalDAV servers
- **Proxy Authentication**: Corporate environments with HTTP proxies

### ICS Sources
- **No Authentication**: Public calendars and feeds
- **Basic Authentication**: Username/password protected ICS files
- **Bearer Token**: API tokens for services
- **Custom Headers**: Complex API authentication schemes

## Configuration Features Demonstrated

### Global Settings
- Output directory configuration
- Date range filtering
- Database storage and change tracking
- Display options (emojis, hashtags, frontmatter, checkboxes)
- Obsidian tasks preset
- Debug logging

### Source Management
- Calendar discovery and auto-detection
- Include/exclude calendar filtering
- Calendar name aliases for cleaner display
- Timeout configuration for remote sources
- Custom HTTP headers

### Multi-Source Benefits
- Global deduplication across all sources
- Source attribution in output
- Parallel processing
- Error isolation (one source failure doesn't stop others)

## Customization Tips

### Security Best Practices
1. **Never commit real credentials** to version control
2. **Use environment variables** for sensitive data:
   ```yaml
   password: ${CALENDAR_PASSWORD}
   client_secret: ${GOOGLE_CLIENT_SECRET}
   ```
3. **Use app-specific passwords** for Apple/iCloud
4. **Generate API tokens** instead of using account passwords
5. **Limit OAuth scopes** to calendar read-only access

### Performance Optimization
1. **Enable database storage** for smart updates:
   ```yaml
   use_database: true
   database_path: ~/.config/caldav2markdown/calendar.db
   ```
2. **Enable server-side filtering** when supported:
   ```yaml
   use_server_side_filtering: true
   ```
3. **Use specific calendar filters** instead of processing everything:
   ```yaml
   include_calendars: ["Work", "Personal"]
   ```
4. **Set appropriate timeouts** for network requests:
   ```yaml
   timeout: 30s  # Reasonable default
   ```

### Display Customization
1. **Use calendar aliases** for cleaner output:
   ```yaml
   calendar_aliases:
     "Very Long Calendar Name": "Short"
   ```
2. **Choose appropriate display options**:
   ```yaml
   obsidian_tasks: true      # One-click Obsidian optimization
   # OR customize individually:
   event_checkboxes: true
   use_due_date_emoji: true
   ignore_descriptions: false
   ```

### Multi-Source Strategy
1. **Organize by purpose**: Work calendars, personal calendars, public feeds
2. **Use descriptive names**: Make it clear what each source provides
3. **Test individually**: Use `-test` flag to verify each source works
4. **Monitor performance**: Watch for slow sources that might need optimization

## Troubleshooting

### Common Issues
1. **Authentication failures**: Check credentials and OAuth tokens
2. **Network timeouts**: Increase timeout values for slow sources
3. **Calendar not found**: Use `-list-calendars` flag to see available calendars
4. **Proxy issues**: Verify proxy settings and authentication

### Debug Mode
Enable detailed logging to troubleshoot issues:
```yaml
trace_web_calls: true
```

### Testing Connections
Test your configuration before running a full sync:
```bash
bin/caldav2markdown -config my-config.yaml -test
```

## Migration from Environment Files

Convert your existing `.env` configuration to YAML:
```bash
bin/caldav2markdown -convert-to-yaml my-config.yaml -config .env
```

The conversion will create a clean YAML file with all your current settings while removing legacy fields.

## Contributing

If you have additional example configurations that would be helpful for others, please consider contributing them to the project. Particularly useful would be examples for:
- Additional calendar systems (Zimbra, Kerio, etc.)
- Industry-specific integrations
- Educational institution setups
- Government/public sector configurations

Remember to sanitize any real credentials before sharing!
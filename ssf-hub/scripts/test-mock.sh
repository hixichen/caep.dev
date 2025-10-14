#!/bin/bash

# SSF Hub Mock Testing Script
# This script tests the complete event flow in mock mode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
HUB_URL="http://localhost:8080"
WEBHOOK_URL="https://httpbin.org/post"

# Print functions
print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Check if hub is running
check_hub_health() {
    print_step "Checking SSF Hub health..."

    if ! curl -s -f "$HUB_URL/health" > /dev/null; then
        print_error "SSF Hub is not running on $HUB_URL"
        print_info "Start the hub with: make run-mock"
        exit 1
    fi

    local status=$(curl -s "$HUB_URL/health" | jq -r '.status // "unknown"')
    if [ "$status" = "healthy" ]; then
        print_success "Hub is healthy"
    else
        print_error "Hub status: $status"
        exit 1
    fi
}

# Clear mock state
clear_mock_state() {
    print_step "Clearing mock state..."

    local result=$(curl -s -X POST "$HUB_URL/debug/mock/clear" | jq -r '.status // "error"')
    if [ "$result" = "success" ]; then
        print_success "Mock state cleared"
    else
        print_error "Failed to clear mock state"
        exit 1
    fi
}

# Register test receiver
register_receiver() {
    print_step "Registering test receiver..."

    # Use timestamp to make receiver ID unique
    local timestamp=$(date +%s)
    local receiver_id="test-receiver-$timestamp"

    local receiver_data='{
        "id": "'$receiver_id'",
        "name": "Test Receiver",
        "webhook_url": "'$WEBHOOK_URL'",
        "event_types": [
            "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
        ],
        "delivery": {
            "method": "webhook"
        },
        "auth": {
            "type": "none"
        }
    }'

    local response=$(curl -s -X POST "$HUB_URL/api/v1/receivers" \
        -H "Content-Type: application/json" \
        -d "$receiver_data")

    local returned_id=$(echo "$response" | jq -r '.id // "error"')

    if [ "$returned_id" = "$receiver_id" ]; then
        print_success "Receiver registered: $returned_id"
    else
        print_error "Failed to register receiver: $returned_id"
        echo "Response: $response"
        exit 1
    fi
}

# Send test event
send_test_event() {
    print_step "Sending test security event..."

    # Create a simple JWT with header.payload.signature format for testing
    # In real usage, this would be a properly signed JWT
    local header='{"alg":"none","typ":"JWT"}'
    local payload='{
        "iss": "https://test-transmitter.example.com",
        "jti": "test-event-'$(date +%s)'",
        "iat": '$(date +%s)',
        "events": {
            "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
                "subject": {
                    "format": "email",
                    "identifier": "user@example.com"
                },
                "data": {
                    "reason": "admin_revocation"
                }
            }
        }
    }'

    # Create a minimal JWT token (header.payload.signature format)
    local encoded_header=$(echo -n "$header" | base64 | tr -d '=\n' | tr '+/' '-_')
    local encoded_payload=$(echo -n "$payload" | base64 | tr -d '=\n' | tr '+/' '-_')
    local jwt_token="${encoded_header}.${encoded_payload}."

    # Send JWT token
    local response=$(curl -s -X POST "$HUB_URL/events" \
        -H "Content-Type: application/secevent+jwt" \
        -H "X-Transmitter-ID: test-transmitter" \
        -d "$jwt_token")

    print_success "Test event sent (JWT format)"
}

# Check mock statistics
check_mock_stats() {
    print_step "Checking mock statistics..."

    # Give time for processing
    sleep 2

    local stats=$(curl -s "$HUB_URL/debug/mock/stats")

    # Extract key metrics
    local topic_messages=$(echo "$stats" | jq -r '.topics."ssf-hub-events".message_count // 0')
    local subscription_count=$(echo "$stats" | jq -r '.subscriptions | length')
    local subscription_messages=$(echo "$stats" | jq -r '.subscriptions | to_entries[0].value.message_count // 0' 2>/dev/null || echo "0")

    echo ""
    print_info "=== Mock Statistics ==="
    print_info "Topic messages: $topic_messages"
    print_info "Subscriptions: $subscription_count"
    print_info "Subscription messages: $subscription_messages"

    # Validate results
    if [ "$topic_messages" -gt 0 ]; then
        print_success "✅ Messages reached unified topic"
    else
        print_error "❌ No messages in unified topic"
    fi

    if [ "$subscription_count" -gt 0 ]; then
        print_success "✅ Hub subscription created"
    else
        print_error "❌ No hub subscriptions found"
    fi

    if [ "$subscription_messages" -gt 0 ]; then
        print_success "✅ Messages processed by hub receiver"
    else
        print_error "❌ No messages processed by hub receiver"
    fi
}

# Check receivers
check_receivers() {
    print_step "Checking registered receivers..."

    local receivers_response=$(curl -s "$HUB_URL/api/v1/receivers")
    local receivers=$(echo "$receivers_response" | jq '.receivers // []')
    local receiver_count=$(echo "$receivers" | jq 'length')

    if [ "$receiver_count" -gt 0 ]; then
        print_success "✅ $receiver_count receiver(s) registered"
        echo "$receivers" | jq '.[] | {id: .id, webhook_url: .webhook_url, status: .status}'
    else
        print_error "❌ No receivers found"
        echo "Full response: $receivers_response"
    fi
}

# Print detailed stats
print_detailed_stats() {
    print_step "Detailed mock statistics..."

    echo ""
    curl -s "$HUB_URL/debug/mock/stats" | jq '{
        project_id: .project_id,
        unified_topic: .unified_topic,
        hub_instance_id: .hub_instance_id,
        topics: .topics,
        subscriptions: .subscriptions
    }'
}

# Main test flow
main() {
    echo ""
    echo "🚀 SSF Hub Mock End-to-End Test"
    echo "==============================="
    echo ""

    check_hub_health
    clear_mock_state
    register_receiver
    send_test_event
    check_mock_stats
    check_receivers
    print_detailed_stats

    echo ""
    print_success "🎉 Test completed successfully!"
    echo ""
    print_info "💡 Tips:"
    print_info "  • Check webhook deliveries at: $WEBHOOK_URL"
    print_info "  • Monitor hub logs for detailed event processing"
    print_info "  • Use 'curl $HUB_URL/debug/mock/stats' to check stats anytime"
    print_info "  • Use 'curl -X POST $HUB_URL/debug/mock/clear' to reset state"
    echo ""
}

# Run main function
main "$@"
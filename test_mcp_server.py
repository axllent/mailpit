#!/usr/bin/env python3
"""
Test MCP Server functionality
"""

import json
import subprocess
import sys
import time

def test_mcp_stdio():
    """Test MCP server via stdio transport"""
    print("Testing MCP Server via stdio transport...")
    
    # Start Mailpit with MCP in stdio mode
    proc = subprocess.Popen(
        ['/tmp/mailpit-test', '--mcp-server', '--mcp-transport', 'stdio'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    # Give it time to initialize
    time.sleep(2)
    
    # Send initialize request
    init_request = {
        "jsonrpc": "2.0",
        "method": "initialize",
        "params": {
            "clientInfo": {
                "name": "test-client",
                "version": "1.0.0"
            }
        },
        "id": 1
    }
    
    try:
        # Send request
        proc.stdin.write(json.dumps(init_request) + '\n')
        proc.stdin.flush()
        
        # Read response with timeout
        proc.stdout.flush()
        response_line = proc.stdout.readline()
        
        if response_line:
            response = json.loads(response_line)
            if "result" in response:
                print("✓ MCP server responded to initialize")
                print(f"  Server: {response['result'].get('serverInfo', {})}")
                return True
            else:
                print("✗ MCP server did not initialize properly")
                print(f"  Response: {response}")
                return False
        else:
            print("✗ No response from MCP server")
            return False
            
    except Exception as e:
        print(f"✗ Error testing MCP: {e}")
        return False
    finally:
        proc.terminate()
        proc.wait()

def test_mcp_tools():
    """Test that MCP tools are registered"""
    print("\nChecking MCP tool registration...")
    
    # This would require a full MCP client implementation
    # For now, we'll just check that the server starts
    proc = subprocess.Popen(
        ['/tmp/mailpit-test', '--mcp-server', '--mcp-transport', 'stdio', '--verbose'],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True
    )
    
    # Give it time to initialize
    time.sleep(2)
    
    # Check logs for tool registration
    proc.terminate()
    output, _ = proc.communicate()
    
    if "server initialized with 4 tools" in output:
        print("✓ MCP tools registered successfully")
        return True
    else:
        print("✗ MCP tools not registered")
        print(f"  Output: {output[:500]}")
        return False

def main():
    print("═══════════════════════════════════════════════════════════════")
    print("                MCP SERVER TEST SUITE")
    print("═══════════════════════════════════════════════════════════════")
    
    tests_passed = 0
    tests_total = 0
    
    # Test 1: stdio transport
    tests_total += 1
    if test_mcp_stdio():
        tests_passed += 1
    
    # Test 2: tool registration
    tests_total += 1
    if test_mcp_tools():
        tests_passed += 1
    
    print("\n═══════════════════════════════════════════════════════════════")
    print(f"Results: {tests_passed}/{tests_total} tests passed")
    
    if tests_passed == tests_total:
        print("✓ ALL MCP TESTS PASSED")
        return 0
    else:
        print("✗ SOME MCP TESTS FAILED")
        return 1

if __name__ == "__main__":
    sys.exit(main())
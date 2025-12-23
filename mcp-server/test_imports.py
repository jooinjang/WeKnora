#!/usr/bin/env python3
"""
Test MCP Imports
"""

try:
    import mcp.types as types

    print("✓ mcp.types imported successfully")
except ImportError as e:
    print(f"✗ mcp.types import failed: {e}")

try:
    from mcp.server import NotificationOptions, Server

    print("✓ mcp.server imported successfully")
except ImportError as e:
    print(f"✗ mcp.server import failed: {e}")

try:
    import mcp.server.stdio

    print("✓ mcp.server.stdio imported successfully")
except ImportError as e:
    print(f"✗ mcp.server.stdio import failed: {e}")

try:
    from mcp.server.models import InitializationOptions

    print("✓ InitializationOptions imported successfully from mcp.server.models")
except ImportError:
    try:
        from mcp import InitializationOptions

        print("✓ InitializationOptions imported successfully from mcp")
    except ImportError as e:
        print(f"✗ InitializationOptions import failed: {e}")

# Check MCP package structure
import mcp

print(f"\nMCP Package Version: {getattr(mcp, '__version__', 'Unknown')}")
print(f"MCP Package Path: {mcp.__file__}")
print(f"MCP Package Contents: {dir(mcp)}")

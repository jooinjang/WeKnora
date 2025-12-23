#!/usr/bin/env python3
"""
WeKnora MCP Server Main Entry Point

This file provides a unified entry point to start the WeKnora MCP server.
Can be run in multiple ways:
1. python main.py
2. python -m weknora_mcp_server
3. weknora-mcp-server (after installation)
"""

import argparse
import asyncio
import os
import sys
from pathlib import Path


def setup_environment():
    """Setup environment and paths"""
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))


def check_dependencies():
    """Check if dependencies are installed"""
    try:
        import mcp
        import requests

        return True
    except ImportError as e:
        print(f"Missing dependency: {e}")
        print("Please run: pip install -r requirements.txt")
        return False


def check_environment_variables():
    """Check environment variable configuration"""
    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    print("=== WeKnora MCP Server Environment Check ===")
    print(f"Base URL: {base_url or 'http://localhost:8080/api/v1 (default)'}")
    print(f"API Key: {'Set' if api_key else 'Not set (warning)'}")

    if not base_url:
        print("Tip: You can set the WEKNORA_BASE_URL environment variable")

    if not api_key:
        print("Warning: It is recommended to set the WEKNORA_API_KEY environment variable")

    print("=" * 40)
    return True


def parse_arguments():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(
        description="WeKnora MCP Server - Model Context Protocol server for WeKnora API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python main.py                    # Start with default configuration
  python main.py --check-only       # Check environment only, don't start server
  python main.py --verbose          # Enable verbose logging

Environment Variables:
  WEKNORA_BASE_URL    WeKnora API base URL (default: http://localhost:8080/api/v1)
  WEKNORA_API_KEY     WeKnora API key
        """,
    )

    parser.add_argument(
        "--check-only", action="store_true", help="Check environment configuration only, don't start server"
    )

    parser.add_argument("--verbose", "-v", action="store_true", help="Enable verbose log output")

    parser.add_argument(
        "--version", action="version", version="WeKnora MCP Server 1.0.0"
    )

    return parser.parse_args()


async def main():
    """Main function"""
    args = parse_arguments()

    setup_environment()

    if not check_dependencies():
        sys.exit(1)

    check_environment_variables()

    if args.check_only:
        print("Environment check completed.")
        return

    if args.verbose:
        import logging

        logging.basicConfig(level=logging.DEBUG)
        print("Verbose logging enabled")

    try:
        print("Starting WeKnora MCP Server...")

        from weknora_mcp_server import run

        await run()

    except ImportError as e:
        print(f"Import error: {e}")
        print("Please ensure all files are in the correct location")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped")
    except Exception as e:
        print(f"Server runtime error: {e}")
        if args.verbose:
            import traceback

            traceback.print_exc()
        sys.exit(1)


def sync_main():
    """Synchronous version of main function for entry_points"""
    asyncio.run(main())


if __name__ == "__main__":
    asyncio.run(main())

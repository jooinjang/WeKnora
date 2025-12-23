import re
from typing import Callable, Dict, List, Match, Pattern, Union

from pydantic import BaseModel, Field


class HeaderTrackerHook(BaseModel):
    """Configuration class for header tracking Hook, supports header recognition in various scenarios"""

    start_pattern: Pattern[str] = Field(
        description="Header start match (regex or string)"
    )
    end_pattern: Pattern[str] = Field(description="Header end match (regex or string)")
    extract_header_fn: Callable[[Match[str]], str] = Field(
        default=lambda m: m.group(0),
        description="Function to extract header content from start match result (default: take the entire matched content)",
    )
    priority: int = Field(default=0, description="Priority (when multiple configs exist, higher priority matches first)")
    case_sensitive: bool = Field(
        default=True, description="Whether case sensitive (only applies when string pattern is provided)"
    )

    def __init__(
        self,
        start_pattern: Union[str, Pattern[str]],
        end_pattern: Union[str, Pattern[str]],
        **kwargs,
    ):
        flags = 0 if kwargs.get("case_sensitive", True) else re.IGNORECASE
        if isinstance(start_pattern, str):
            start_pattern = re.compile(start_pattern, flags | re.DOTALL)
        if isinstance(end_pattern, str):
            end_pattern = re.compile(end_pattern, flags | re.DOTALL)
        super().__init__(
            start_pattern=start_pattern,
            end_pattern=end_pattern,
            **kwargs,
        )


# Initialize header Hook configuration (provide default config: support Markdown tables, code blocks)
DEFAULT_CONFIGS = [
    # Code block configuration (starts with ```, ends with ```)
    # HeaderTrackerHook(
    #     # Code block start (supports language specification)
    #     start_pattern=r"^\s*```(\w+).*(?!```$)",
    #     # Code block end
    #     end_pattern=r"^\s*```.*$",
    #     extract_header_fn=lambda m: f"```{m.group(1)}" if m.group(1) else "```",
    #     priority=20,  # Code block priority is higher than table
    #     case_sensitive=True,
    # ),
    # Markdown table configuration (header with underline)
    HeaderTrackerHook(
        # Header row + separator line
        start_pattern=r"^\s*(?:\|[^|\n]*)+[\r\n]+\s*(?:\|\s*:?-{3,}:?\s*)+\|?[\r\n]+$",
        # Empty line or non-table content
        end_pattern=r"^\s*$|^\s*[^|\s].*$",
        priority=15,
        case_sensitive=False,
    ),
]
DEFAULT_CONFIGS.sort(key=lambda x: -x.priority)


# Define Hook state data structure
class HeaderTracker(BaseModel):
    """State class for header tracking Hook"""

    header_hook_configs: List[HeaderTrackerHook] = Field(default=DEFAULT_CONFIGS)
    active_headers: Dict[int, str] = Field(default_factory=dict)
    ended_headers: set[int] = Field(default_factory=set)

    def update(self, split: str) -> Dict[int, str]:
        """Detect header start/end in current split, update Hook state"""
        new_headers: Dict[int, str] = {}

        # 1. Check for header end markers
        for config in self.header_hook_configs:
            if config.priority in self.active_headers and config.end_pattern.search(
                split
            ):
                self.ended_headers.add(config.priority)
                del self.active_headers[config.priority]

        # 2. Check for new header start markers (only process those that are not active and not ended)
        for config in self.header_hook_configs:
            if (
                config.priority not in self.active_headers
                and config.priority not in self.ended_headers
            ):
                match = config.start_pattern.search(split)
                if match:
                    header = config.extract_header_fn(match)
                    self.active_headers[config.priority] = header
                    new_headers[config.priority] = header

        # 3. Check if all active headers have ended (clear end markers)
        if not self.active_headers:
            self.ended_headers.clear()

        return new_headers

    def get_headers(self) -> str:
        """Get concatenated text of all currently active headers (sorted by priority)"""
        # Sort headers in descending order of priority
        sorted_headers = sorted(self.active_headers.items(), key=lambda x: -x[0])
        return (
            "\n".join([header for _, header in sorted_headers])
            if sorted_headers
            else ""
        )

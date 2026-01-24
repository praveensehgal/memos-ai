import { useLocation } from "react-router-dom";
import { type MemoFilter, stringifyFilters, useMemoFilterContext } from "@/contexts/MemoFilterContext";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";
import { Routes } from "@/router";
import { useMemoViewContext } from "../MemoView/MemoViewContext";

interface TagPillsProps {
  tags: string[];
  className?: string;
}

// Extract tags from memo content
export const extractTagsFromContent = (content: string): string[] => {
  const tagRegex = /#([\p{L}\p{N}\p{S}_/-]+)/gu;
  const tags: string[] = [];
  let match: RegExpExecArray | null;

  while ((match = tagRegex.exec(content)) !== null) {
    const tag = match[1];
    if (tag && tag.length <= 100 && !tags.includes(tag)) {
      tags.push(tag);
    }
  }

  return tags;
};

export const TagPills: React.FC<TagPillsProps> = ({ tags, className }) => {
  const { parentPage } = useMemoViewContext();
  const location = useLocation();
  const navigateTo = useNavigateTo();
  const { getFiltersByFactor, removeFilter, addFilter } = useMemoFilterContext();

  if (tags.length === 0) {
    return null;
  }

  const handleTagClick = (e: React.MouseEvent, tag: string) => {
    e.stopPropagation();

    // If the tag is clicked in a memo detail page, navigate to the memo list page
    if (location.pathname.startsWith("/m")) {
      const pathname = parentPage || Routes.ROOT;
      const searchParams = new URLSearchParams();
      searchParams.set("filter", stringifyFilters([{ factor: "tagSearch", value: tag }]));
      navigateTo(`${pathname}?${searchParams.toString()}`);
      return;
    }

    const isActive = getFiltersByFactor("tagSearch").some((filter: MemoFilter) => filter.value === tag);
    if (isActive) {
      removeFilter((f: MemoFilter) => f.factor === "tagSearch" && f.value === tag);
    } else {
      // Remove all existing tag filters first, then add the new one
      removeFilter((f: MemoFilter) => f.factor === "tagSearch");
      addFilter({
        factor: "tagSearch",
        value: tag,
      });
    }
  };

  return (
    <div
      className={cn(
        "w-full flex flex-wrap gap-1.5 px-3 py-2 -mx-3 mt-2 rounded-md bg-muted/50",
        className,
      )}
    >
      {tags.map((tag) => {
        const isActive = getFiltersByFactor("tagSearch").some((filter: MemoFilter) => filter.value === tag);
        return (
          <button
            key={tag}
            type="button"
            onClick={(e) => handleTagClick(e, tag)}
            className={cn(
              "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium transition-colors",
              "bg-primary/10 text-primary hover:bg-primary/20",
              isActive && "bg-primary text-primary-foreground hover:bg-primary/90",
            )}
          >
            #{tag}
          </button>
        );
      })}
    </div>
  );
};

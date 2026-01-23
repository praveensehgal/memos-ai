import { CheckIcon, LoaderIcon, SparklesIcon, XIcon } from "lucide-react";
import type { FC } from "react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatTagForDisplay, useAITagSuggestions } from "@/hooks/useAITagSuggestions";
import { cn } from "@/lib/utils";
import type { TagSuggestion } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";

interface AITagSuggestionsProps {
  content: string;
  existingTags: string[];
  onTagSelect: (tag: string) => void;
  disabled?: boolean;
}

/**
 * AI-powered tag suggestions component for the memo editor.
 * Shows a popover with suggested tags based on memo content.
 */
export const AITagSuggestions: FC<AITagSuggestionsProps> = ({ content, existingTags, onTagSelect, disabled }) => {
  const t = useTranslate();
  const [open, setOpen] = useState(false);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());

  const { mutate: suggestTags, data, isPending, error, reset } = useAITagSuggestions();

  const handleOpen = (isOpen: boolean) => {
    setOpen(isOpen);
    if (isOpen && !data && !isPending) {
      // Fetch suggestions when opening
      suggestTags({
        content,
        existingTags,
        maxTags: 5,
      });
    }
    if (!isOpen) {
      // Reset selections when closing
      setSelectedTags(new Set());
    }
  };

  const handleRefresh = () => {
    reset();
    setSelectedTags(new Set());
    suggestTags({
      content,
      existingTags,
      maxTags: 5,
    });
  };

  const handleTagClick = (tag: string) => {
    const newSelected = new Set(selectedTags);
    if (newSelected.has(tag)) {
      newSelected.delete(tag);
    } else {
      newSelected.add(tag);
    }
    setSelectedTags(newSelected);
  };

  const handleApplySelected = () => {
    selectedTags.forEach((tag) => {
      onTagSelect(tag);
    });
    setSelectedTags(new Set());
    setOpen(false);
  };

  const suggestions = data?.suggestions || [];
  const isConfigured = data?.isConfigured ?? true;
  const errorMessage = data?.errorMessage || (error instanceof Error ? error.message : undefined);

  // Check if content is too short for meaningful suggestions
  const contentTooShort = content.trim().length < 10;

  return (
    <Popover open={open} onOpenChange={handleOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-2 text-muted-foreground hover:text-foreground"
              disabled={disabled || contentTooShort}
            >
              <SparklesIcon className="size-4" />
              <span className="sr-only">{t("editor.ai-suggest-tags")}</span>
            </Button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent>
          <p>{contentTooShort ? t("editor.ai-suggest-tags-content-too-short") : t("editor.ai-suggest-tags")}</p>
        </TooltipContent>
      </Tooltip>

      <PopoverContent className="w-72 p-3" align="start">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-medium flex items-center gap-1.5">
              <SparklesIcon className="size-4 text-primary" />
              {t("editor.ai-tag-suggestions")}
            </h4>
            {suggestions.length > 0 && (
              <Button variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={handleRefresh} disabled={isPending}>
                {t("common.refresh")}
              </Button>
            )}
          </div>

          {/* Loading state */}
          {isPending && (
            <div className="flex items-center justify-center py-4">
              <LoaderIcon className="size-5 animate-spin text-muted-foreground" />
              <span className="ml-2 text-sm text-muted-foreground">{t("editor.ai-generating-tags")}</span>
            </div>
          )}

          {/* Error state */}
          {!isPending && errorMessage && (
            <div className="flex items-center gap-2 p-2 rounded-md bg-destructive/10 text-destructive text-sm">
              <XIcon className="size-4 shrink-0" />
              <span>{errorMessage}</span>
            </div>
          )}

          {/* Not configured state */}
          {!isPending && !isConfigured && !errorMessage && (
            <div className="text-sm text-muted-foreground text-center py-3">
              {t("editor.ai-not-configured")}
            </div>
          )}

          {/* Suggestions list */}
          {!isPending && suggestions.length > 0 && (
            <>
              <div className="flex flex-wrap gap-1.5">
                {suggestions.map((suggestion: TagSuggestion) => (
                  <TagBadge
                    key={suggestion.tag}
                    suggestion={suggestion}
                    isSelected={selectedTags.has(suggestion.tag)}
                    onClick={() => handleTagClick(suggestion.tag)}
                  />
                ))}
              </div>

              {/* Apply button */}
              {selectedTags.size > 0 && (
                <Button size="sm" className="w-full" onClick={handleApplySelected}>
                  <CheckIcon className="size-4 mr-1" />
                  {t("editor.ai-apply-tags", { count: selectedTags.size })}
                </Button>
              )}

              <p className="text-xs text-muted-foreground">{t("editor.ai-click-to-select")}</p>
            </>
          )}

          {/* No suggestions */}
          {!isPending && !errorMessage && isConfigured && suggestions.length === 0 && data && (
            <div className="text-sm text-muted-foreground text-center py-3">{t("editor.ai-no-suggestions")}</div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};

interface TagBadgeProps {
  suggestion: TagSuggestion;
  isSelected: boolean;
  onClick: () => void;
}

const TagBadge: FC<TagBadgeProps> = ({ suggestion, isSelected, onClick }) => {
  const confidenceLevel = suggestion.confidence >= 0.7 ? "high" : suggestion.confidence >= 0.4 ? "medium" : "low";

  return (
    <Badge
      variant={isSelected ? "default" : "outline"}
      className={cn(
        "cursor-pointer transition-all hover:scale-105",
        suggestion.isExisting && "opacity-50",
        !isSelected && confidenceLevel === "high" && "border-primary/50",
      )}
      onClick={onClick}
    >
      {isSelected && <CheckIcon className="size-3 mr-0.5" />}
      {formatTagForDisplay(suggestion.tag)}
      {suggestion.isExisting && <span className="ml-1 text-[10px]">(exists)</span>}
    </Badge>
  );
};

export default AITagSuggestions;

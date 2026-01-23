import { create } from "@bufbuild/protobuf";
import { useMutation } from "@tanstack/react-query";
import { memoServiceClient } from "@/connect";
import type { SuggestTagsResponse, TagSuggestion } from "@/types/proto/api/v1/memo_service_pb";
import { SuggestTagsRequestSchema } from "@/types/proto/api/v1/memo_service_pb";

// Query keys factory for AI tag suggestions
export const aiTagKeys = {
  all: ["ai-tags"] as const,
  suggestions: (content: string) => [...aiTagKeys.all, "suggestions", content] as const,
};

export interface SuggestTagsParams {
  content: string;
  existingTags?: string[];
  maxTags?: number;
}

export interface AITagSuggestionResult {
  suggestions: TagSuggestion[];
  isConfigured: boolean;
  errorMessage?: string;
}

/**
 * Hook for fetching AI-generated tag suggestions for memo content.
 * Uses useMutation since this is an on-demand operation, not cached data.
 *
 * @returns Mutation object with suggestTags function and state
 *
 * @example
 * ```tsx
 * const { mutate: suggestTags, data, isPending, error } = useAITagSuggestions();
 *
 * // Request tag suggestions
 * suggestTags({ content: "My memo about React and TypeScript" });
 *
 * // Access results
 * if (data?.suggestions) {
 *   data.suggestions.forEach(s => console.log(s.tag, s.confidence));
 * }
 * ```
 */
export function useAITagSuggestions() {
  return useMutation({
    mutationFn: async (params: SuggestTagsParams): Promise<AITagSuggestionResult> => {
      const request = create(SuggestTagsRequestSchema, {
        content: params.content,
        existingTags: params.existingTags || [],
        maxTags: params.maxTags || 5,
      });

      try {
        const response: SuggestTagsResponse = await memoServiceClient.suggestTags(request);
        return {
          suggestions: response.suggestions,
          isConfigured: true,
        };
      } catch (error) {
        // Handle specific error cases
        if (error instanceof Error) {
          const message = error.message.toLowerCase();

          // LLM not configured
          if (message.includes("not configured") || message.includes("failedprecondition")) {
            return {
              suggestions: [],
              isConfigured: false,
              errorMessage: "AI tag suggestions require LLM to be configured in settings.",
            };
          }

          // Auto-tagging disabled
          if (message.includes("disabled")) {
            return {
              suggestions: [],
              isConfigured: true,
              errorMessage: "AI tag suggestions are disabled in settings.",
            };
          }

          // Rate limited
          if (message.includes("rate limit") || message.includes("resourceexhausted")) {
            return {
              suggestions: [],
              isConfigured: true,
              errorMessage: "Rate limit exceeded. Please try again later.",
            };
          }
        }

        // Re-throw unexpected errors
        throw error;
      }
    },
  });
}

/**
 * Extract tag strings from suggestions, optionally filtering by confidence threshold.
 *
 * @param suggestions - Array of TagSuggestion objects
 * @param minConfidence - Minimum confidence threshold (0-1), defaults to 0
 * @returns Array of tag strings
 */
export function extractTagsFromSuggestions(suggestions: TagSuggestion[], minConfidence = 0): string[] {
  return suggestions.filter((s) => s.confidence >= minConfidence).map((s) => s.tag);
}

/**
 * Format a tag for display, adding the # prefix if not present.
 *
 * @param tag - Tag string (with or without #)
 * @returns Tag string with # prefix
 */
export function formatTagForDisplay(tag: string): string {
  return tag.startsWith("#") ? tag : `#${tag}`;
}

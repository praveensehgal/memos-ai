import { type FC, useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { useTranslate } from "@/utils/i18n";
import { validationService } from "../services";
import { useEditorContext } from "../state";
import InsertMenu from "../Toolbar/InsertMenu";
import VisibilitySelector from "../Toolbar/VisibilitySelector";
import type { EditorToolbarProps } from "../types";
import { AITagSuggestions } from "./AITagSuggestions";

/**
 * Extract tags from memo content using a simple regex pattern.
 * Matches #tag patterns where tag can contain letters, numbers, underscores, hyphens, and slashes.
 */
function extractTagsFromContent(content: string): string[] {
  const tagRegex = /#([\p{L}\p{N}_\-/]+)/gu;
  const tags: string[] = [];
  let match: RegExpExecArray | null;
  while ((match = tagRegex.exec(content)) !== null) {
    tags.push(match[1]);
  }
  return [...new Set(tags)]; // Remove duplicates
}

export const EditorToolbar: FC<EditorToolbarProps> = ({ onSave, onCancel, memoName, editorRef }) => {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();
  const { valid } = validationService.canSave(state);

  const isSaving = state.ui.isLoading.saving;

  // Extract existing tags from content for AI suggestions
  const existingTags = useMemo(() => extractTagsFromContent(state.content), [state.content]);

  const handleLocationChange = (location: typeof state.metadata.location) => {
    dispatch(actions.setMetadata({ location }));
  };

  const handleToggleFocusMode = () => {
    dispatch(actions.toggleFocusMode());
  };

  const handleVisibilityChange = (visibility: typeof state.metadata.visibility) => {
    dispatch(actions.setMetadata({ visibility }));
  };

  // Handle AI tag selection - insert tag at the end of content
  const handleTagSelect = useCallback(
    (tag: string) => {
      if (editorRef?.current) {
        // Format tag with # prefix if not present
        const formattedTag = tag.startsWith("#") ? tag : `#${tag}`;
        // Add space before tag if content doesn't end with space or newline
        const content = state.content;
        const needsSpace = content.length > 0 && !content.endsWith(" ") && !content.endsWith("\n");
        const textToInsert = needsSpace ? ` ${formattedTag} ` : `${formattedTag} `;
        editorRef.current.insertText(textToInsert);
      }
    },
    [editorRef, state.content],
  );

  return (
    <div className="w-full flex flex-row justify-between items-center mb-2">
      <div className="flex flex-row justify-start items-center gap-1">
        <InsertMenu
          isUploading={state.ui.isLoading.uploading}
          location={state.metadata.location}
          onLocationChange={handleLocationChange}
          onToggleFocusMode={handleToggleFocusMode}
          memoName={memoName}
        />
        <AITagSuggestions
          content={state.content}
          existingTags={existingTags}
          onTagSelect={handleTagSelect}
          disabled={isSaving || !editorRef?.current}
        />
      </div>

      <div className="flex flex-row justify-end items-center gap-2">
        <VisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />

        {onCancel && (
          <Button variant="ghost" onClick={onCancel} disabled={isSaving}>
            {t("common.cancel")}
          </Button>
        )}

        <Button onClick={onSave} disabled={!valid || isSaving}>
          {isSaving ? t("editor.saving") : t("editor.save")}
        </Button>
      </div>
    </div>
  );
};

import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useInstance } from "@/contexts/InstanceContext";
import { handleError } from "@/lib/error";
import {
  InstanceSetting_Key,
  InstanceSetting_LLMSetting,
  InstanceSetting_LLMSettingSchema,
  InstanceSetting_LLMSetting_LLMProvider,
  InstanceSetting_LLMAnthropicConfigSchema,
  InstanceSetting_LLMGeminiConfigSchema,
  InstanceSetting_LLMOllamaConfigSchema,
  InstanceSetting_LLMOpenAIConfigSchema,
  InstanceSettingSchema,
} from "@/types/proto/api/v1/instance_service_pb";
import { useTranslate } from "@/utils/i18n";
import SettingGroup from "./SettingGroup";
import SettingRow from "./SettingRow";
import SettingSection from "./SettingSection";

const MASKED_KEY = "***masked***";

const providerOptions = [
  { value: InstanceSetting_LLMSetting_LLMProvider.LLM_PROVIDER_UNSPECIFIED, label: "None" },
  { value: InstanceSetting_LLMSetting_LLMProvider.OPENAI, label: "OpenAI" },
  { value: InstanceSetting_LLMSetting_LLMProvider.ANTHROPIC, label: "Anthropic" },
  { value: InstanceSetting_LLMSetting_LLMProvider.GEMINI, label: "Google Gemini" },
  { value: InstanceSetting_LLMSetting_LLMProvider.OLLAMA, label: "Ollama (Local)" },
];

const AISettingsSection = () => {
  const t = useTranslate();
  const { llmSetting: originalSetting, updateSetting, fetchSetting } = useInstance();
  const [llmSetting, setLLMSetting] = useState<InstanceSetting_LLMSetting>(originalSetting);

  const updatePartialSetting = (partial: Partial<InstanceSetting_LLMSetting>) => {
    const newSetting = create(InstanceSetting_LLMSettingSchema, {
      ...llmSetting,
      ...partial,
    });
    setLLMSetting(newSetting);
  };

  const handleProviderChange = (value: string) => {
    const provider = Number(value) as InstanceSetting_LLMSetting_LLMProvider;
    updatePartialSetting({ provider });
  };

  const handleUpdateSetting = async () => {
    try {
      await updateSetting(
        create(InstanceSettingSchema, {
          name: `instance/settings/${InstanceSetting_Key[InstanceSetting_Key.LLM]}`,
          value: {
            case: "llmSetting",
            value: llmSetting,
          },
        }),
      );
      await fetchSetting(InstanceSetting_Key.LLM);
      toast.success(t("message.update-succeed"));
    } catch (error: unknown) {
      await handleError(error, toast.error, {
        context: "Update AI settings",
      });
    }
  };

  const isProviderConfigured = llmSetting.provider !== InstanceSetting_LLMSetting_LLMProvider.LLM_PROVIDER_UNSPECIFIED;

  return (
    <SettingSection>
      <SettingGroup title={t("setting.ai-settings.title")}>
        <SettingRow label={t("setting.ai-settings.provider")} description={t("setting.ai-settings.provider-description")}>
          <Select value={String(llmSetting.provider)} onValueChange={handleProviderChange}>
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {providerOptions.map((option) => (
                <SelectItem key={option.value} value={String(option.value)}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SettingRow>
      </SettingGroup>

      {llmSetting.provider === InstanceSetting_LLMSetting_LLMProvider.OPENAI && (
        <SettingGroup title={t("setting.ai-settings.openai-config")} showSeparator>
          <SettingRow label={t("setting.ai-settings.api-key")} tooltip={t("setting.ai-settings.api-key-tooltip")}>
            <Input
              type="password"
              className="w-64"
              placeholder="sk-..."
              value={llmSetting.openaiConfig?.apiKey === MASKED_KEY ? "" : llmSetting.openaiConfig?.apiKey || ""}
              onChange={(e) =>
                updatePartialSetting({
                  openaiConfig: create(InstanceSetting_LLMOpenAIConfigSchema, {
                    ...llmSetting.openaiConfig,
                    apiKey: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.base-url")} description={t("setting.ai-settings.base-url-description")}>
            <Input
              className="w-64"
              placeholder="https://api.openai.com/v1"
              value={llmSetting.openaiConfig?.baseUrl || ""}
              onChange={(e) =>
                updatePartialSetting({
                  openaiConfig: create(InstanceSetting_LLMOpenAIConfigSchema, {
                    ...llmSetting.openaiConfig,
                    baseUrl: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.model")}>
            <Input
              className="w-48"
              placeholder="gpt-4o-mini"
              value={llmSetting.openaiConfig?.defaultModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  openaiConfig: create(InstanceSetting_LLMOpenAIConfigSchema, {
                    ...llmSetting.openaiConfig,
                    defaultModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.embedding-model")} description={t("setting.ai-settings.embedding-model-description")}>
            <Input
              className="w-48"
              placeholder="text-embedding-3-small"
              value={llmSetting.openaiConfig?.embeddingModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  openaiConfig: create(InstanceSetting_LLMOpenAIConfigSchema, {
                    ...llmSetting.openaiConfig,
                    embeddingModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
        </SettingGroup>
      )}

      {llmSetting.provider === InstanceSetting_LLMSetting_LLMProvider.ANTHROPIC && (
        <SettingGroup title={t("setting.ai-settings.anthropic-config")} showSeparator>
          <SettingRow label={t("setting.ai-settings.api-key")} tooltip={t("setting.ai-settings.api-key-tooltip")}>
            <Input
              type="password"
              className="w-64"
              placeholder="sk-ant-..."
              value={llmSetting.anthropicConfig?.apiKey === MASKED_KEY ? "" : llmSetting.anthropicConfig?.apiKey || ""}
              onChange={(e) =>
                updatePartialSetting({
                  anthropicConfig: create(InstanceSetting_LLMAnthropicConfigSchema, {
                    ...llmSetting.anthropicConfig,
                    apiKey: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.base-url")} description={t("setting.ai-settings.base-url-description")}>
            <Input
              className="w-64"
              placeholder="https://api.anthropic.com"
              value={llmSetting.anthropicConfig?.baseUrl || ""}
              onChange={(e) =>
                updatePartialSetting({
                  anthropicConfig: create(InstanceSetting_LLMAnthropicConfigSchema, {
                    ...llmSetting.anthropicConfig,
                    baseUrl: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.model")}>
            <Input
              className="w-48"
              placeholder="claude-3-5-sonnet-20241022"
              value={llmSetting.anthropicConfig?.defaultModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  anthropicConfig: create(InstanceSetting_LLMAnthropicConfigSchema, {
                    ...llmSetting.anthropicConfig,
                    defaultModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
        </SettingGroup>
      )}

      {llmSetting.provider === InstanceSetting_LLMSetting_LLMProvider.GEMINI && (
        <SettingGroup title={t("setting.ai-settings.gemini-config")} showSeparator>
          <SettingRow label={t("setting.ai-settings.api-key")} tooltip={t("setting.ai-settings.api-key-tooltip")}>
            <Input
              type="password"
              className="w-64"
              placeholder="AIza..."
              value={llmSetting.geminiConfig?.apiKey === MASKED_KEY ? "" : llmSetting.geminiConfig?.apiKey || ""}
              onChange={(e) =>
                updatePartialSetting({
                  geminiConfig: create(InstanceSetting_LLMGeminiConfigSchema, {
                    ...llmSetting.geminiConfig,
                    apiKey: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.model")}>
            <Input
              className="w-48"
              placeholder="gemini-1.5-flash"
              value={llmSetting.geminiConfig?.defaultModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  geminiConfig: create(InstanceSetting_LLMGeminiConfigSchema, {
                    ...llmSetting.geminiConfig,
                    defaultModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
        </SettingGroup>
      )}

      {llmSetting.provider === InstanceSetting_LLMSetting_LLMProvider.OLLAMA && (
        <SettingGroup title={t("setting.ai-settings.ollama-config")} showSeparator>
          <SettingRow label={t("setting.ai-settings.host")} description={t("setting.ai-settings.host-description")}>
            <Input
              className="w-64"
              placeholder="http://localhost:11434"
              value={llmSetting.ollamaConfig?.host || ""}
              onChange={(e) =>
                updatePartialSetting({
                  ollamaConfig: create(InstanceSetting_LLMOllamaConfigSchema, {
                    ...llmSetting.ollamaConfig,
                    host: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.model")}>
            <Input
              className="w-48"
              placeholder="llama3.2"
              value={llmSetting.ollamaConfig?.defaultModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  ollamaConfig: create(InstanceSetting_LLMOllamaConfigSchema, {
                    ...llmSetting.ollamaConfig,
                    defaultModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.embedding-model")} description={t("setting.ai-settings.embedding-model-description")}>
            <Input
              className="w-48"
              placeholder="nomic-embed-text"
              value={llmSetting.ollamaConfig?.embeddingModel || ""}
              onChange={(e) =>
                updatePartialSetting({
                  ollamaConfig: create(InstanceSetting_LLMOllamaConfigSchema, {
                    ...llmSetting.ollamaConfig,
                    embeddingModel: e.target.value,
                  }),
                })
              }
            />
          </SettingRow>
        </SettingGroup>
      )}

      {isProviderConfigured && (
        <SettingGroup title={t("setting.ai-settings.features")} showSeparator>
          <SettingRow label={t("setting.ai-settings.auto-tagging")} description={t("setting.ai-settings.auto-tagging-description")}>
            <Switch
              checked={llmSetting.enableAutoTagging}
              onCheckedChange={(checked) => updatePartialSetting({ enableAutoTagging: checked })}
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.auto-summary")} description={t("setting.ai-settings.auto-summary-description")}>
            <Switch
              checked={llmSetting.enableAutoSummary}
              onCheckedChange={(checked) => updatePartialSetting({ enableAutoSummary: checked })}
            />
          </SettingRow>
          <SettingRow label={t("setting.ai-settings.semantic-search")} description={t("setting.ai-settings.semantic-search-description")}>
            <Switch
              checked={llmSetting.enableSemanticSearch}
              onCheckedChange={(checked) => updatePartialSetting({ enableSemanticSearch: checked })}
            />
          </SettingRow>
        </SettingGroup>
      )}

      <div className="w-full flex justify-end">
        <Button disabled={isEqual(llmSetting, originalSetting)} onClick={handleUpdateSetting}>
          {t("common.save")}
        </Button>
      </div>
    </SettingSection>
  );
};

export default AISettingsSection;

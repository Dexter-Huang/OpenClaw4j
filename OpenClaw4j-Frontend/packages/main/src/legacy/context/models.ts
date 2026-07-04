import { createContext } from 'react';
import type { LegacyModelItem } from '../services/prompt';

export const ModelsContext = createContext<{
  modelNameMap: Record<string, string>;
  models: LegacyModelItem[];
  setModels: (models: LegacyModelItem[]) => void;
}>({
  modelNameMap: {},
  models: [],
  setModels: (models: LegacyModelItem[]) => {},
});

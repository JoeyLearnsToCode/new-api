import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SettingsState {
  // Theme（嵌入 new-api 后由 AppInitializer 跟随全局主题同步）
  theme: 'light' | 'dark';
  setTheme: (theme: 'light' | 'dark') => void;

  // Editor
  editorFontSize: number;
  editorLineHeight: number;
  setEditorSettings: (fontSize: number, lineHeight: number) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'light',
      setTheme: (theme) => set({ theme }),

      editorFontSize: 14,
      editorLineHeight: 1.5,
      setEditorSettings: (fontSize, lineHeight) =>
        set({ editorFontSize: fontSize, editorLineHeight: lineHeight }),
    }),
    {
      name: 'prompt-studio-settings',
    },
  ),
);

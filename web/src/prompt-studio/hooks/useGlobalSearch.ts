import { useCallback, useEffect, useRef, useState } from 'react';
import { useGlobalSearchStore } from '@/prompt-studio/store/globalSearchStore';
import { useProjectStore } from '@/prompt-studio/store/projectStore';
import { useVersionStore } from '@/prompt-studio/store/versionStore';
import type { Version } from '@/prompt-studio/models/Version';

export const useGlobalSearch = () => {
  const {
    query,
    matches,
    currentIndex,
    total,
    isActive,
    fuzzy,
    setQuery,
    executeSearch,
    nextMatch,
    prevMatch,
    clearSearch,
    toggleFuzzy,
  } = useGlobalSearchStore();

  const { projects } = useProjectStore();
  const { versions, loadAllVersions } = useVersionStore();
  const [allVersions, setAllVersions] = useState<Version[]>([]);

  useEffect(() => {
    if (!isActive) return;
    // 全局搜索需要跨项目比对，从服务端一次性取回全部版本
    loadAllVersions()
      .then((v) => setAllVersions(v))
      .catch(() => {});
  }, [isActive, loadAllVersions]);

  const prevFuzzy = useRef(fuzzy);
  useEffect(() => {
    if (prevFuzzy.current !== fuzzy && query.trim()) {
      const versionsToSearch = allVersions.length > 0 ? allVersions : versions;
      executeSearch(projects, versionsToSearch, query);
    }
    prevFuzzy.current = fuzzy;
  }, [fuzzy, query, projects, versions, allVersions, executeSearch]);

  const handleQueryChange = useCallback(
    (newQuery: string) => {
      setQuery(newQuery);
      const versionsToSearch = allVersions.length > 0 ? allVersions : versions;
      executeSearch(projects, versionsToSearch, newQuery);
    },
    [projects, versions, allVersions, setQuery, executeSearch],
  );

  const handleClear = useCallback(() => {
    clearSearch();
  }, [clearSearch]);

  const getCurrentMatchId = useCallback(() => {
    if (currentIndex < 0 || currentIndex >= matches.length) {
      return null;
    }
    return matches[currentIndex];
  }, [matches, currentIndex]);

  return {
    query,
    matches,
    currentIndex,
    total,
    isActive,
    fuzzy,
    handleQueryChange,
    handleNext: nextMatch,
    handlePrev: prevMatch,
    handleClear,
    getCurrentMatchId,
    handleToggleFuzzy: toggleFuzzy,
  };
};

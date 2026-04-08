"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { User, WorkspaceInfo } from "./types";

interface AuthState {
  readonly token: string | null;
  readonly user: User | null;
  readonly workspaces: WorkspaceInfo[];
  readonly activeWorkspaceId: string | null;
  readonly isLoading: boolean;
}

interface AuthActions {
  setAuth(token: string, user: User, workspaces: WorkspaceInfo[]): void;
  switchWorkspace(workspaceId: string): void;
  logout(): void;
}

type AuthContextValue = AuthState & AuthActions;

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({
  children,
}: {
  readonly children: React.ReactNode;
}) {
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([]);
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(
    null
  );
  const [isLoading, setIsLoading] = useState(true);

  // Restore from localStorage on mount
  useEffect(() => {
    const savedToken = localStorage.getItem("token");
    const savedUser = localStorage.getItem("user");
    const savedWorkspaces = localStorage.getItem("workspaces");
    const savedWsId = localStorage.getItem("workspace_id");

    if (savedToken && savedUser) {
      try {
        setToken(savedToken);
        setUser(JSON.parse(savedUser));
        setWorkspaces(savedWorkspaces ? JSON.parse(savedWorkspaces) : []);
        setActiveWorkspaceId(savedWsId);
      } catch {
        // Corrupted storage — clear it
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        localStorage.removeItem("workspaces");
        localStorage.removeItem("workspace_id");
      }
    }
    setIsLoading(false);
  }, []);

  const setAuth = useCallback(
    (newToken: string, newUser: User, newWorkspaces: WorkspaceInfo[]) => {
      setToken(newToken);
      setUser(newUser);
      setWorkspaces(newWorkspaces);

      const defaultWsId = newWorkspaces[0]?.id ?? null;
      setActiveWorkspaceId(defaultWsId);

      localStorage.setItem("token", newToken);
      localStorage.setItem("user", JSON.stringify(newUser));
      localStorage.setItem("workspaces", JSON.stringify(newWorkspaces));
      if (defaultWsId) {
        localStorage.setItem("workspace_id", defaultWsId);
      }
    },
    []
  );

  const switchWorkspace = useCallback(
    (workspaceId: string) => {
      setActiveWorkspaceId(workspaceId);
      localStorage.setItem("workspace_id", workspaceId);
    },
    []
  );

  const logout = useCallback(() => {
    setToken(null);
    setUser(null);
    setWorkspaces([]);
    setActiveWorkspaceId(null);
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    localStorage.removeItem("workspaces");
    localStorage.removeItem("workspace_id");
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user,
      workspaces,
      activeWorkspaceId,
      isLoading,
      setAuth,
      switchWorkspace,
      logout,
    }),
    [
      token,
      user,
      workspaces,
      activeWorkspaceId,
      isLoading,
      setAuth,
      switchWorkspace,
      logout,
    ]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}

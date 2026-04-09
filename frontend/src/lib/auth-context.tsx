"use client";

import {
  createContext,
  useCallback,
  useContext,
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

function loadStoredAuth(): {
  token: string | null;
  user: User | null;
  workspaces: WorkspaceInfo[];
  activeWorkspaceId: string | null;
} {
  if (typeof window === "undefined") {
    return { token: null, user: null, workspaces: [], activeWorkspaceId: null };
  }
  const savedToken = localStorage.getItem("token");
  const savedUser = localStorage.getItem("user");
  const savedWorkspaces = localStorage.getItem("workspaces");
  const savedWsId = localStorage.getItem("workspace_id");

  if (savedToken && savedUser) {
    try {
      return {
        token: savedToken,
        user: JSON.parse(savedUser),
        workspaces: savedWorkspaces ? JSON.parse(savedWorkspaces) : [],
        activeWorkspaceId: savedWsId,
      };
    } catch {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      localStorage.removeItem("workspaces");
      localStorage.removeItem("workspace_id");
    }
  }
  return { token: null, user: null, workspaces: [], activeWorkspaceId: null };
}

export function AuthProvider({
  children,
}: {
  readonly children: React.ReactNode;
}) {
  const [stored] = useState(loadStoredAuth);
  const [token, setToken] = useState<string | null>(stored.token);
  const [user, setUser] = useState<User | null>(stored.user);
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>(
    stored.workspaces
  );
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(
    stored.activeWorkspaceId
  );
  const isLoading = false;

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

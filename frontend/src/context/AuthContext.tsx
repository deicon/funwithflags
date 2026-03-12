import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";

import { apiFetch, login as apiLogin, logout as apiLogout, refresh as apiRefresh } from "@/api/client";

interface AuthUser {
  username: string;
  role: string;
}

interface AuthState {
  accessToken: string;
  refreshToken: string;
  accessTokenExpiresAt: string;
  refreshTokenExpiresAt: string;
  user: AuthUser;
}

interface AuthContextValue {
  user: AuthUser | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  authorizedFetch: typeof apiFetch;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const STORAGE_KEY = "funwithflags.auth";

function loadStoredAuth(): AuthState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as AuthState;
    if (!parsed.accessToken || !parsed.refreshToken) {
      return null;
    }
    return parsed;
  } catch (error) {
    return null;
  }
}

function storeAuth(state: AuthState | null) {
  if (!state) {
    localStorage.removeItem(STORAGE_KEY);
    return;
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function isTokenExpired(isoTimestamp: string): boolean {
  const expiresAt = Date.parse(isoTimestamp);
  return Number.isNaN(expiresAt) || expiresAt <= Date.now();
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState | null>(() => {
    const stored = loadStoredAuth();
    if (!stored) {
      return null;
    }
    if (isTokenExpired(stored.refreshTokenExpiresAt)) {
      return null;
    }
    return stored;
  });

  const saveState = useCallback((next: AuthState | null) => {
    setState(next);
    storeAuth(next);
  }, []);

  const refreshSession = useCallback(async () => {
    if (!state) {
      return;
    }
    try {
      const response = await apiRefresh(state.refreshToken);
      saveState({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        accessTokenExpiresAt: response.accessTokenExpiresAt,
        refreshTokenExpiresAt: response.refreshTokenExpiresAt,
        user: response.user
      });
    } catch (error) {
      console.error("failed to refresh session", error);
      saveState(null);
      throw error;
    }
  }, [saveState, state]);

  useEffect(() => {
    if (!state) {
      return;
    }
    if (isTokenExpired(state.accessTokenExpiresAt)) {
      refreshSession().catch(() => undefined);
      return;
    }

    const expiresAt = Date.parse(state.accessTokenExpiresAt);
    const refreshDelay = Math.max(expiresAt - Date.now() - 60_000, 5_000);
    const timer = window.setTimeout(() => {
      refreshSession().catch(() => undefined);
    }, refreshDelay);
    return () => window.clearTimeout(timer);
  }, [refreshSession, state]);

  const login = useCallback(
    async (username: string, password: string) => {
      const response = await apiLogin(username, password);
      saveState({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        accessTokenExpiresAt: response.accessTokenExpiresAt,
        refreshTokenExpiresAt: response.refreshTokenExpiresAt,
        user: response.user
      });
    },
    [saveState]
  );

  const logout = useCallback(async () => {
    if (state?.refreshToken) {
      try {
        await apiLogout(state.refreshToken);
      } catch (error) {
        console.warn("logout failed", error);
      }
    }
    saveState(null);
  }, [saveState, state?.refreshToken]);

  const authorizedFetch: typeof apiFetch = useCallback(
    async (path, options) => {
      if (!state) {
        throw new Error("not authenticated");
      }
      return apiFetch(path, {
        ...options,
        token: state.accessToken
      });
    },
    [state]
  );

  const value: AuthContextValue = useMemo(
    () => ({
      user: state?.user ?? null,
      accessToken: state?.accessToken ?? null,
      refreshToken: state?.refreshToken ?? null,
      isAuthenticated: Boolean(state),
      login,
      logout,
      refreshSession,
      authorizedFetch
    }),
    [authorizedFetch, login, logout, refreshSession, state]
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

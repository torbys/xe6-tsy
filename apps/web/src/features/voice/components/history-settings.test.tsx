import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { saveAuthSession } from "../lib/auth-session";
import { HistoryRecentSettings, HistorySettings } from "./history-settings";

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  });
}

describe("HistorySettings", () => {
  beforeEach(() => {
    localStorage.clear();
    saveAuthSession({
      account: {
        id: "acc-history",
        kind: "anonymous",
        created_at: "2026-08-01T00:00:00Z",
      },
      tokens: {
        access_token: "access-history",
        refresh_token: "refresh-history",
        expires_at: "2099-08-01T00:00:00Z",
      },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("/voice-sessions?") && !url.includes("/turns")) {
          return jsonResponse({
            sessions: [
              {
                id: "vs-history-20260804",
                account_id: "acc-history",
                status: "ended",
                created_at: "2026-08-04T08:00:00Z",
                started_at: "2026-08-04T08:00:00Z",
                ended_at: "2026-08-04T08:12:00Z",
              },
            ],
            next_cursor: null,
          });
        }
        if (url.includes("/vs-history-20260804/turns")) {
          return jsonResponse({
            items: [
              {
                id: "turn-history-1",
                session_id: "vs-history-20260804",
                source_language: "zh-CN",
                target_language: "en-US",
                source_text: "欢迎来到主会场。",
                translated_text: "Welcome to the main venue.",
                sequence_no: 1,
                created_at: "2026-08-04T08:01:00Z",
              },
            ],
            next_cursor: null,
          });
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  it("opens a labeled transcript for the selected historical session", async () => {
    const onExit = vi.fn();
    render(<HistorySettings onExit={onExit} />);

    const sessionButton = await screen.findByRole("button", {
      name: /查看.*历史记录/,
    });
    expect(screen.getByText(/12 分钟/)).toBeInTheDocument();
    fireEvent.click(sessionButton);

    expect(await screen.findByText("欢迎来到主会场。")).toBeInTheDocument();
    expect(screen.getByText("Welcome to the main venue.")).toBeInTheDocument();
    expect(screen.getByText(/会话 vs-histo/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "返回设置" }));
    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("keeps recent settings items to metadata until opened", async () => {
    const onOpen = vi.fn();
    render(<HistoryRecentSettings onOpen={onOpen} />);

    const sessionButton = await screen.findByRole("button", {
      name: /打开.*历史会话/,
    });
    expect(screen.queryByText("Welcome to the main venue.")).not.toBeInTheDocument();
    fireEvent.click(sessionButton);

    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(onOpen.mock.calls[0][0].id).toBe("vs-history-20260804");
  });
});

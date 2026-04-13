"use client";

interface GlobalErrorProps {
  readonly error: Error & { digest?: string };
  readonly reset: () => void;
}

export default function GlobalError({ error, reset }: GlobalErrorProps) {
  return (
    <html lang="en">
      <body
        className="antialiased"
        style={{
          backgroundColor: "#fbfcfd",
          color: "#090f12",
          fontFamily:
            "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            minHeight: "100vh",
          }}
        >
          <div
            style={{
              backgroundColor: "#ffffff",
              borderRadius: "12px",
              border: "1px solid rgba(239, 68, 68, 0.2)",
              padding: "40px",
              textAlign: "center",
              maxWidth: "448px",
              width: "100%",
              margin: "0 16px",
            }}
          >
            <div
              style={{
                width: "48px",
                height: "48px",
                borderRadius: "50%",
                backgroundColor: "rgba(239, 68, 68, 0.1)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                margin: "0 auto 16px",
              }}
            >
              <svg
                width="24"
                height="24"
                fill="none"
                stroke="#ef4444"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.07 16.5c-.77.833.192 2.5 1.732 2.5z"
                />
              </svg>
            </div>
            <h2
              style={{
                fontSize: "18px",
                fontWeight: 600,
                marginBottom: "8px",
              }}
            >
              Application Error
            </h2>
            <p
              style={{
                fontSize: "14px",
                color: "#5f6b7a",
                marginBottom: "24px",
              }}
            >
              {error.message ||
                "A critical error occurred. Please reload the page."}
            </p>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: "12px",
              }}
            >
              <button
                onClick={reset}
                style={{
                  padding: "8px 16px",
                  fontSize: "14px",
                  fontWeight: 500,
                  color: "#ffffff",
                  backgroundColor: "#5694af",
                  border: "none",
                  borderRadius: "8px",
                  cursor: "pointer",
                }}
              >
                Try again
              </button>
              <button
                onClick={() => (window.location.href = "/")}
                style={{
                  padding: "8px 16px",
                  fontSize: "14px",
                  fontWeight: 500,
                  color: "#5f6b7a",
                  backgroundColor: "#f0f2f7",
                  border: "none",
                  borderRadius: "8px",
                  cursor: "pointer",
                }}
              >
                Reload
              </button>
            </div>
          </div>
        </div>
      </body>
    </html>
  );
}

import claudeLogo from "@/assets/mcp-clients/claude.svg";
import cursorLogo from "@/assets/mcp-clients/cursor.svg";
import openaiLogo from "@/assets/mcp-clients/openai.svg";

export type McpClient = {
  name: string;
  logo: string;
  instructions: "chatgptStep" | "claudeStep" | "commandStep" | "cursorStep";
  signIn: "oauthStep" | "codexSignInStep" | "claudeCodeSignInStep";
  docs: string;
  /** Terminal clients take a whole command; the others take the URL in their settings. */
  command?: (url: string) => string;
};

export const clients: McpClient[] = [
  {
    name: "ChatGPT",
    logo: openaiLogo,
    instructions: "chatgptStep",
    signIn: "oauthStep",
    docs: "https://developers.openai.com/api/docs/guides/custom-mcp-server",
  },
  {
    name: "Claude",
    logo: claudeLogo,
    instructions: "claudeStep",
    signIn: "oauthStep",
    docs: "https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp",
  },
  {
    name: "Codex",
    logo: openaiLogo,
    instructions: "commandStep",
    signIn: "codexSignInStep",
    docs: "https://developers.openai.com/codex/mcp",
    command: (url) => `codex mcp add koi --url ${url}`,
  },
  {
    name: "Claude Code",
    logo: claudeLogo,
    instructions: "commandStep",
    signIn: "claudeCodeSignInStep",
    docs: "https://code.claude.com/docs/en/mcp",
    command: (url) => `claude mcp add --transport http koi ${url}`,
  },
  {
    name: "Cursor",
    logo: cursorLogo,
    instructions: "cursorStep",
    signIn: "oauthStep",
    docs: "https://cursor.com/docs/context/mcp",
  },
];

export function ClientLogo({ logo, className }: { logo: string; className: string }) {
  return (
    <img
      src={logo}
      alt=""
      className={logo === claudeLogo ? className : `${className} dark:invert`}
    />
  );
}

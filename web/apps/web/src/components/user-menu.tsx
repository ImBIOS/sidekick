import { Button } from "@sidekick/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@sidekick/ui/components/dropdown-menu";
import { Link, useNavigate } from "@tanstack/react-router";

import { authClient } from "@/lib/auth-client";

export default function UserMenu() {
  const navigate = useNavigate();
  const { data: session, isPending } = authClient.useSession();

  if (isPending) {
    // Static placeholder: same footprint, no pulse (avoids a blinking grey box on every load).
    return <span className="block h-9 w-40" aria-hidden="true" />;
  }

  if (!session) {
    return (
      <>
        <Link to="/login" search={{ mode: "signin" }} className="px-2 text-sm text-zinc-400 transition-colors hover:text-white">
          Log in
        </Link>
        <Link to="/login">
          <Button className="rounded-full bg-white px-5 text-black hover:bg-zinc-200">Sign up</Button>
        </Link>
      </>
    );
  }

  return (
    <>
      <Link to="/dashboard">
        <Button className="rounded-full bg-white px-5 text-black hover:bg-zinc-200">Dashboard</Button>
      </Link>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="outline" />}>
          {session.user.name}
        </DropdownMenuTrigger>
        <DropdownMenuContent className="bg-card">
          <DropdownMenuGroup>
            <DropdownMenuLabel>My Account</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem>{session.user.email}</DropdownMenuItem>
            <DropdownMenuItem
              variant="destructive"
              onClick={() => {
                authClient.signOut({
                  fetchOptions: {
                    onSuccess: () => {
                      navigate({
                        to: "/",
                      });
                    },
                  },
                });
              }}
            >
              Sign Out
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}

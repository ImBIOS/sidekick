import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@sidekick/ui/components/button";
import { Card } from "@sidekick/ui/components/card";
import { Input } from "@sidekick/ui/components/input";
import { Label } from "@sidekick/ui/components/label";
import { useTRPC } from "@/utils/trpc";

export const Route = createFileRoute("/_auth/settings")({
  component: RouteComponent,
});

function RouteComponent() {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const [keyName, setKeyName] = useState("sidekick CLI");
  const [newSecret, setNewSecret] = useState<string | null>(null);

  const orgs = useQuery(trpc.organization.list.queryOptions());
  const activeOrg = useQuery(trpc.organization.active.queryOptions());
  const keys = useQuery(trpc.apiKeys.list.queryOptions());

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: trpc.organization.list.queryKey() }).then(() => {
      queryClient.invalidateQueries({ queryKey: trpc.organization.active.queryKey() });
      queryClient.invalidateQueries({ queryKey: trpc.apiKeys.list.queryKey() });
    });

  const setActive = useMutation(
    trpc.organization.setActive.mutationOptions({
      onSuccess: () => {
        toast.success("Active organization updated");
        invalidate();
      },
      onError: (e) => toast.error(e.message),
    }),
  );

  const createKey = useMutation(
    trpc.apiKeys.create.mutationOptions({
      onSuccess: (data) => {
        setNewSecret(data.key);
        setKeyName("sidekick CLI");
        toast.success("API key created — copy it now, it won't be shown again");
        queryClient.invalidateQueries({ queryKey: trpc.apiKeys.list.queryKey() });
      },
      onError: (e) => toast.error(e.message),
    }),
  );

  const revokeKey = useMutation(
    trpc.apiKeys.revoke.mutationOptions({
      onSuccess: () => {
        toast.success("API key revoked");
        queryClient.invalidateQueries({ queryKey: trpc.apiKeys.list.queryKey() });
      },
      onError: (e) => toast.error(e.message),
    }),
  );

  return (
    <div className="mx-auto w-full max-w-2xl space-y-6 p-6">
      <h1 className="text-3xl font-bold">Settings</h1>

      <Card className="space-y-3 p-6">
        <h2 className="text-xl font-semibold">Organization</h2>
        <p className="text-sm text-muted-foreground">
          Remote builds, simulators, and emulators are billed and isolated per organization. Your
          default org was created at signup — the CLI uses it with zero setup.
        </p>
        {activeOrg.data && (
          <p className="text-sm">
            Active: <span className="font-mono font-semibold">{activeOrg.data.name}</span>{" "}
            <span className="font-mono text-muted-foreground">({activeOrg.data.slug})</span>
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          {orgs.data?.map((org) => (
            <Button
              key={org.id}
              variant={org.id === activeOrg.data?.id ? "default" : "outline"}
              size="sm"
              disabled={setActive.isPending}
              onClick={() => setActive.mutate({ organizationId: org.id })}
            >
              {org.name}
            </Button>
          ))}
        </div>
      </Card>

      <Card className="space-y-3 p-6">
        <h2 className="text-xl font-semibold">API keys</h2>
        <p className="text-sm text-muted-foreground">
          Keys are scoped to the active org. Run{" "}
          <code className="font-mono">export SIDEKICK_TOKEN=&lt;key&gt;</code> (or{" "}
          <code className="font-mono">sidekick login</code>) so agents can use remote builds and
          simulators.
        </p>

        {newSecret && (
          <div className="space-y-2 rounded-md border border-dashed p-3">
            <Label>New key (shown once)</Label>
            <div className="flex gap-2">
              <Input readOnly value={newSecret} className="font-mono" />
              <Button
                variant="outline"
                onClick={() => {
                  void navigator.clipboard.writeText(newSecret);
                  toast.success("Copied to clipboard");
                }}
              >
                Copy
              </Button>
              <Button variant="ghost" onClick={() => setNewSecret(null)}>
                Dismiss
              </Button>
            </div>
          </div>
        )}

        <div className="flex gap-2">
          <Input
            value={keyName}
            onChange={(e) => setKeyName(e.target.value)}
            placeholder="Key name"
            maxLength={32}
          />
          <Button
            disabled={createKey.isPending || keyName.trim().length === 0}
            onClick={() => createKey.mutate({ name: keyName.trim() })}
          >
            Create key
          </Button>
        </div>

        <ul className="space-y-2">
          {keys.data?.map((key) => (
            <li key={key.id} className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="font-medium">{key.name ?? "Unnamed key"}</p>
                <p className="font-mono text-xs text-muted-foreground">
                  {key.prefix ?? ""}…{key.start?.slice(-4)} · id {key.id.slice(0, 8)}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                disabled={revokeKey.isPending}
                onClick={() => revokeKey.mutate({ keyId: key.id })}
              >
                Revoke
              </Button>
            </li>
          ))}
          {keys.data?.length === 0 && (
            <p className="text-sm text-muted-foreground">No keys yet — create one above.</p>
          )}
        </ul>
      </Card>
    </div>
  );
}

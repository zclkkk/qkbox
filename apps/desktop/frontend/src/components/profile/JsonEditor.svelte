<script lang="ts">
  import { json } from "@codemirror/lang-json";
  import { basicSetup, EditorView } from "codemirror";
  import { onDestroy, onMount } from "svelte";

  let {
    value,
    onchange
  }: {
    value: string;
    onchange: (value: string) => void;
  } = $props();

  let host: HTMLDivElement;
  let view: EditorView | null = null;

  onMount(() => {
    view = new EditorView({
      doc: value,
      extensions: [
        basicSetup,
        json(),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onchange(update.state.doc.toString());
          }
        })
      ],
      parent: host
    });
  });

  $effect(() => {
    if (!view) {
      return;
    }
    const current = view.state.doc.toString();
    if (value !== current) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: value }
      });
    }
  });

  onDestroy(() => {
    view?.destroy();
  });
</script>

<div class="json-editor" bind:this={host}></div>

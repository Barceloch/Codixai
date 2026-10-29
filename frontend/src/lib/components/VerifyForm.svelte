<script lang="ts">
	import { onMount } from 'svelte'
	import Header from './Header.svelte'
	import { i18n } from '../lib/i18n.svelte'
	import { VerifyAsync, CancelAudit, ChooseFile, Export, isBridgeAvailable, onVerifyEvent } from '../lib/bridge'
	import { ui } from '../lib/wailsjs/go/models'
	import type { domain } from '../lib/wailsjs/go/models'

	let bridgeAvailable = $state(isBridgeAvailable())
	let root = $state('')
	let base = $state('')
	let claimsPath = $state('')
	let reportPath = $state('')
	let running = $state(false)
	let progress = $state<ui.VerifyProgress | null>(null)
	let view = $state<ui.VerifyView | null>(null)
	let error = $state('')
	let notice = $state('')
	let runID = $state('')
	let filters = $state({
		state: '',
		severity: '',
		path: '',
		sort: 'id',
		direction: 'asc' as 'asc' | 'desc'
	})
	let selectedKey = $state('')

	const claims = $derived(view?.document?.results ?? [])
	const filtered = $derived(
		view?.document?.results
			.filter((r) => {
				if (filters.state && r.status !== filters.state) return false
				if (filters.severity && r.claim?.severity !== filters.severity) return false
				if (filters.path && !r.claim?.paths?.some((p: string) => p.toLowerCase().includes(filters.path.toLowerCase()))) return false
				return true
			})
			.sort((a, b) => {
				const dir = filters.direction === 'desc' ? -1 : 1
				if (filters.sort === 'id') return a.claim.id.localeCompare(b.claim.id) * dir
				if (filters.sort === 'state') return a.status.localeCompare(b.status) * dir
				if (filters.sort === 'severity') return 0
				return 0
			})
	)

	const selected = $derived(
		view?.document?.results.find((r, i) => `${r.claim.id}:${i}` === selectedKey) ?? null
	)

	function newRunID(): string {
		if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
			return `run-${crypto.randomUUID()}`
		}
		return `run-${Date.now()}`
	}

	function describeError(cause: unknown): string {
		if (cause instanceof Error && cause.message) return cause.message
		if (typeof cause === 'string' && cause.trim()) return cause
		return i18n.t('verify.error.generic')
	}

	async function chooseRoot(): Promise<void> {
		error = ''
		try {
			const path = await ChooseFile(i18n.t('verify.chooseRepositoryTitle'))
			if (path) root = path
		} catch (cause) {
			error = describeError(cause)
		}
	}

	async function chooseClaims(): Promise<void> {
		error = ''
		try {
			const path = await ChooseFile(i18n.t('verify.chooseClaimsTitle'), ['.json'])
			if (path) claimsPath = path
		} catch (cause) {
			error = describeError(cause)
		}
	}

	async function chooseReport(): Promise<void> {
		error = ''
		try {
			const path = await ChooseFile(i18n.t('verify.chooseReportTitle'), ['.xml', '.json'])
			if (path) reportPath = path
		} catch (cause) {
			error = describeError(cause)
		}
	}

	async function run(): Promise<void> {
		if (running) return
		if (!root.trim()) { error = i18n.t('verify.error.noRoot'); return }
		if (!base.trim()) { error = i18n.t('verify.error.noBase'); return }
		if (!claimsPath.trim()) { error = i18n.t('verify.error.noClaims'); return }

		error = ''
		notice = ''
		progress = null
		view = null
		selectedKey = ''
		runID = `run-${Date.now()}`
		running = true

		try {
			const request = new ui.VerifyRequest({
				root: root.trim(),
				base: base.trim(),
				claims_path: claimsPath,
				run_id: `run-${Date.now()}`,
			})
			await VerifyAsync(request, `run-${Date.now()}`)
		} catch (cause) {
			running = false
			progress = null
			error = describeError(cause)
		}
	}

	async function cancel(): Promise<void> {
		if (!runID) return
		try {
			await CancelAudit(runID)
		} catch (cause) {
			error = describeError(cause)
		}
	}

	async function exportReport(format: 'json' | 'sarif'): Promise<void> {
		if (!view) return
		error = ''
		notice = ''
		try {
			const path = await Export(
				new ui.ExportRequest({
					run_id: view.run_id,
					format,
					suggested_name: `codixai-verify.${format}`,
					title: i18n.t('export.saveTitle'),
				}),
			)
			notice = path
				? i18n.t('export.saved', { path })
				: i18n.t('export.cancelled')
		} catch (cause) {
			error = describeError(cause)
		}
	}

	function selectResult(result: domain.ClaimResult, index: number): void {
		selectedKey = `${result.claim.id}:${index}`
	}

	function applyEvent(event: ui.VerifyEvent): void {
		if (!event || event.run_id !== runID) return
		if (event.kind === 'progress' && event.progress) {
			progress = event.progress
			return
		}
		if (event.kind === 'done' && event.view) {
			running = false
			progress = null
			view = event.view
			selectedKey = ''
			return
		}
		if (event.kind === 'cancelled') {
			running = false
			progress = null
			return
		}
		if (event.kind === 'error') {
			running = false
			progress = null
			error = event.error || 'Unknown error'
		}
	}

	async function prefill(): Promise<void> {
		try {
			const path = await Root()
			if (path && !root) root = path
		} catch {
			// ignore
		}
	}

	$effect(() => {
		bridgeAvailable = isBridgeAvailable()
		if (!bridgeAvailable) return
		const stop = onVerifyEvent(applyEvent)
		prefill()
		return stop
	})
</script>

<svelte:head>
	<title>{i18n.t('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen flex-col">
	<Header />

	<main class="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 px-4 py-6">
		<div class="card-glass glass-surface">
			<div class="card-body">
				<h2 class="card-title text-primary-color">{i18n.t('verify.title')}</h2>

				<div class="grid gap-4 md:grid-cols-3">
					<div class="form-control">
						<label class="label text-xs text-primary-color/60">{i18n.t('verify.repository')}</label>
						<div class="flex gap-2">
							<input
								type="text"
								class="input input-bordered w-full"
								bind:value={root}
								placeholder={i18n.t('verify.rootPlaceholder')}
							/>
							<button class="btn-glass btn-sm" onclick={chooseRoot} disabled={running}>
								{i18n.t('verify.choose')}
							</button>
						</div>
					</div>

					<div class="form-control">
						<label class="label text-xs text-primary-color/60">{i18n.t('verify.baseRevision')}</label>
						<input
							type="text"
							class="input input-bordered"
							bind:value={base}
							placeholder={i18n.t('verify.basePlaceholder')}
							disabled={running}
						/>
					</div>
				</div>

				<div class="grid gap-4 md:grid-cols-2">
					<div class="form-control">
						<label class="label text-xs text-primary-color/60">{i18n.t('verify.claimsFile')}</label>
						<div class="flex gap-2">
							<input
								type="text"
								class="input input-bordered w-full"
								bind:value={claimsPath}
								placeholder={i18n.t('verify.claimsPlaceholder')}
							/>
							<button class="btn-glass btn-sm" onclick={chooseClaims} disabled={running}>
								{i18n.t('verify.choose')}
							</button>
						</div>
					</div>

					<div class="form-control">
						<label class="label text-xs text-primary-color/60">{i18n.t('verify.reportFile')}</label>
						<div class="flex gap-2">
							<input
								type="text"
								class="input input-bordered w-full"
								bind:value={reportPath}
								placeholder={i18n.t('verify.reportPlaceholder')}
							/>
							<button class="btn-glass btn-sm" onclick={chooseReport} disabled={running}>
								{i18n.t('verify.choose')}
							</button>
						</div>
					</div>
				</div>

				<div class="flex gap-2">
					<button class="btn-primary-glass" onclick={run} disabled={running}>
						{running ? i18n.t('verify.running') : i18n.t('verify.run')}
					</button>
					<button class="btn-glass" onclick={cancel} disabled={!running}>
						{i18n.t('verify.cancel')}
					</button>
				</div>

				{#if error}
					<div class="alert alert-error-glass" role="alert">
						<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m-2 2l2 2m-2 2l-2 2m2 2l-2 2m2 2l2 2m-2 2l-2 2m2 2l2 2" /></svg>
						<span class="text-primary-color">{error}</span>
					</div>
				{/if}

				{#if notice}
					<div class="alert alert-success-glass" role="alert">
						<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0" /></svg>
						<span class="text-primary-color">{notice}</span>
					</div>
				{/if}

				{#if running}
					<div class="progress progress-glass" role="progressbar" aria-valuenow={progress?.total || 0} aria-valuemin="0" aria-valuemax="100">
						<div class="progress-glass-bar" style="width: {progress?.total ? Math.min(100, (progress.contradicted + progress.verified + progress.unresolved) / progress.total * 100) : 0}%"></div>
					</div>
					<p class="text-sm text-base-content/70 text-primary-color/60">{progress?.phase}</p>
				{/if}
			</div>
		</div>

		{#if notice}
			<div class="alert alert-success-glass" role="alert">
				<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0" /></svg>
				<span class="text-primary-color">{notice}</span>
			</div>
		{/if}

		{#if view}
			<div class="card-glass glass-surface mt-4">
				<div class="card-body">
					<div class="flex flex-wrap items-center justify-between gap-2 mb-4">
						<h2 class="card-title text-primary-color">{i18n.t('verify.results')}</h2>
						<div class="flex flex-wrap gap-2">
							<button class="btn-glass btn-sm" onclick={() => exportReport('json')}>{i18n.t('export.json')}</button>
							<button class="btn-glass btn-sm" onclick={() => exportReport('sarif')}>{i18n.t('export.sarif')}</button>
						</div>
					</div>

					<div class="overflow-x-auto">
						<table class="table-glass">
							<thead>
								<tr>
									<th onclick={() => { filters.sort = 'id'; filters.direction = filters.sort === 'id' && filters.direction === 'asc' ? 'desc' : 'asc' }} class="cursor-pointer text-primary-color">{i18n.t('verify.claimId')}</th>
									<th onclick={() => { filters.sort = 'state'; filters.direction = filters.sort === 'state' && filters.direction === 'asc' ? 'desc' : 'asc' }} class="cursor-pointer text-primary-color">{i18n.t('verify.state')}</th>
									<th onclick={() => { filters.sort = 'severity'; filters.direction = filters.sort === 'severity' && filters.direction === 'asc' ? 'desc' : 'asc' }} class="cursor-pointer text-primary-color">{i18n.t('verify.severity')}</th>
									<th class="text-primary-color">{i18n.t('verify.claim')}</th>
								</tr>
							</thead>
							<tbody>
								{#each filtered as result, index}
									<tr class:selected={selectedKey === `${result.claim.id}:${index}`} onclick={() => selectResult(result, index)}>
										<td class="font-mono text-xs text-primary-color">{result.claim.id}</td>
										<td>
											<span class="badge badge-sm" class:badge-success-glass={result.status === 'verified'} class:badge-error-glass={result.status === 'contradicted'} class:badge-warning-glass={result.status === 'unresolved'}>
												{result.status}
											</span>
										</td>
										<td></td>
										<td class="text-primary-color/70">{result.claim.type}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					{#if selected}
						<div class="divider my-4 border-border-dark"></div>
						<div class="prose prose-sm max-w-none text-primary-color">
							<h4>{i18n.t('verify.detail')}</h4>
							<p><strong>{i18n.t('verify.claim')}:</strong> {selected.claim.type}</p>
							<p><strong>{i18n.t('verify.paths')}:</strong> {selected.claim.paths.join(', ')}</p>
							<p><strong>{i18n.t('verify.message')}:</strong> {selected.message}</p>
							{#if selected.evidence?.length}
								<h5>{i18n.t('verify.evidence')}</h5>
								<ul>
									{#each selected.evidence as ev}
										<li>{ev.kind}: {ev.ref} - {ev.detail}</li>
									{/each}
								</ul>
							{/if}
							{#if selected.limitations?.length}
								<h5>{i18n.t('verify.limitations')}</h5>
								<ul>
									{#each selected.limitations as lim}
										<li>{lim}</li>
									{/each}
								</ul>
							{/if}
						</div>
					{/if}

					<div class="flex flex-wrap gap-2 mt-4">
						<button class="btn-glass btn-sm" onclick={() => exportReport('json')}>{i18n.t('export.json')}</button>
						<button class="btn-glass btn-sm" onclick={() => exportReport('sarif')}>{i18n.t('export.sarif')}</button>
					</div>
				</div>
			</div>
		{/if}
	</main>

	<footer class="border-t border-border-dark bg-base-100/80 backdrop-blur-lg px-4 py-3">
		<p class="mx-auto max-w-7xl text-xs text-base-content/60 text-primary-color/60">{i18n.t('footer.note')}</p>
	</footer>
</div>
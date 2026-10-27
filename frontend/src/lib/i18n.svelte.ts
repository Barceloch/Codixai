// The minimal i18n of the audit interface.
//
// The module is a Svelte 5 rune store: the components read i18n.locale and
// the t() function reads it too, so changing the language re-renders every
// text. The locale is persisted in localStorage and the default follows the
// language of the system.
//
// The values of the document (rule identifiers, paths, messages,
// recommendations, limitations) are data of the analyzed repository and are
// never translated: only the labels of the interface are.

export type Locale = 'en' | 'es'

const STORAGE_KEY = 'codixai.locale'

const en: Record<string, string> = {
  'app.title': 'Codixai',
  'app.subtitle': 'Audit the changes of a repository. The tool only reads it.',
  'app.localOnly': 'Local only',

  'header.language': 'Language',
  'header.theme': 'Theme',
  'header.themeSystem': 'System',
  'header.themeLight': 'Light',
  'header.themeDark': 'Dark',

  'audit.title': 'Audit',
  'audit.repository': 'Repository root',
  'audit.repositoryPlaceholder': '/path/to/repository',
  'audit.chooseRepository': 'Choose…',
  'audit.chooseRepositoryTitle': 'Select the repository to audit',
  'audit.origin': 'Origin',
  'audit.base': 'Base revision',
  'audit.basePlaceholder': 'main',
  'audit.run': 'Run audit',
  'audit.running': 'Auditing…',
  'audit.cancel': 'Cancel',
  'audit.idle': 'No audit yet.',
  'audit.idleHint':
    'Choose the repository and the origin, then run the audit.',

  'audit.progress.validating': 'Validating the request…',
  'audit.progress.collecting': 'Collecting the changes…',
  'audit.progress.collected': 'Collected {count} changed file(s).',
  'audit.progress.analyzing': 'Applying the rule catalog…',
  'audit.progress.analyzed': 'The catalog produced {count} finding(s).',
  'audit.progress.done': 'Audit completed.',
  'audit.progress.cancelled': 'Audit cancelled.',

  'audit.error.title': 'The audit failed',
  'audit.error.bridge':
    'The window runtime is not available. Start the interface with "codixai ui".',
  'audit.error.generic': 'The operation could not be completed.',

  'origin.worktree': 'Worktree (uncommitted changes)',
  'origin.staged': 'Staging area',
  'origin.base': 'Base revision',

  'summary.title': 'Summary',
  'summary.root': 'Repository',
  'summary.origin': 'Origin',
  'summary.base': 'Base',
  'summary.changedFiles': 'Changed files',
  'summary.findings': 'Findings',
  'summary.failOn': 'Fail on',
  'summary.failed': 'The findings reach the failure threshold.',
  'summary.passed': 'No finding reaches the failure threshold.',
  'summary.known': 'Known by the baseline',
  'summary.suppressed': 'Suppressed by the configuration',
  'summary.excluded': 'Excluded files',
  'summary.none': 'none',

  'findings.title': 'Findings',
  'findings.count': '{count} finding(s)',
  'findings.empty': 'No findings.',
  'findings.emptyHint':
    'The absence of findings does not prove that the change is correct.',
  'findings.noResults': 'No finding matches the filters.',
  'findings.noResultsHint': 'Clear the filters to see the whole document.',

  'filters.title': 'Filters',
  'filters.rule': 'Rule',
  'filters.severity': 'Severity',
  'filters.epistemic': 'Epistemic status',
  'filters.language': 'Language',
  'filters.path': 'Path',
  'filters.pathPlaceholder': 'Filter by path…',
  'filters.sort': 'Sort',
  'filters.all': 'All',
  'filters.clear': 'Clear filters',
  'filters.sortPath': 'Path',
  'filters.sortSeverity': 'Severity',
  'filters.sortRule': 'Rule',
  'filters.sortEpistemic': 'Epistemic status',
  'filters.sortLanguage': 'Language',
  'filters.ascending': 'Ascending',
  'filters.descending': 'Descending',

  'table.severity': 'Severity',
  'table.rule': 'Rule',
  'table.epistemic': 'Epistemic',
  'table.language': 'Language',
  'table.location': 'Location',
  'table.message': 'Message',

  'detail.title': 'Finding detail',
  'detail.empty': 'Select a finding to read its evidence.',
  'detail.message': 'Message',
  'detail.location': 'Location',
  'detail.ruleVersion': 'Rule version',
  'detail.evidence': 'Evidence',
  'detail.evidenceKind': 'Kind',
  'detail.evidenceRef': 'Reference',
  'detail.hunk': 'Diff hunk',
  'detail.hunkEmpty': 'The change set does not hold the hunk of this evidence.',
  'detail.support': 'Support',
  'detail.supportLevel': 'Support level',
  'detail.method': 'Method',
  'detail.limitations': 'Limitations',
  'detail.recommendation': 'Recommendation',
  'detail.revision': 'Revision',
  'detail.base': 'Base',
  'detail.head': 'Head',
  'detail.none': 'None declared.',

  'severity.info': 'Info',
  'severity.warning': 'Warning',
  'severity.error': 'Error',

  'epistemic.observed': 'Observed',
  'epistemic.inferred': 'Inferred',
  'epistemic.unknown': 'Unknown',

  'support.none': 'None',
  'support.syntactic': 'Syntactic',
  'support.resolved': 'Resolved',

  'export.title': 'Export',
  'export.json': 'Export JSON',
  'export.sarif': 'Export SARIF',
  'export.saveTitle': 'Save the audit report',
  'export.saved': 'Report written to {path}.',
  'export.cancelled': 'The export was cancelled.',

  'footer.note':
    'Codixai reads the repository: it never modifies it, never executes its code and never accesses the network.',
}

const es: Record<string, string> = {
  'app.title': 'Codixai',
  'app.subtitle': 'Audita los cambios de un repositorio. La herramienta solo lo lee.',
  'app.localOnly': 'Solo local',

  'header.language': 'Idioma',
  'header.theme': 'Tema',
  'header.themeSystem': 'Sistema',
  'header.themeLight': 'Claro',
  'header.themeDark': 'Oscuro',

  'audit.title': 'Auditoría',
  'audit.repository': 'Raíz del repositorio',
  'audit.repositoryPlaceholder': '/ruta/al/repositorio',
  'audit.chooseRepository': 'Elegir…',
  'audit.chooseRepositoryTitle': 'Selecciona el repositorio que quieres auditar',
  'audit.origin': 'Origen',
  'audit.base': 'Revisión base',
  'audit.basePlaceholder': 'main',
  'audit.run': 'Ejecutar auditoría',
  'audit.running': 'Auditando…',
  'audit.cancel': 'Cancelar',
  'audit.idle': 'Todavía no hay auditoría.',
  'audit.idleHint':
    'Elige el repositorio y el origen, y ejecuta la auditoría.',

  'audit.progress.validating': 'Validando la solicitud…',
  'audit.progress.collecting': 'Recopilando los cambios…',
  'audit.progress.collected': 'Se recopilaron {count} archivo(s) cambiado(s).',
  'audit.progress.analyzing': 'Aplicando el catálogo de reglas…',
  'audit.progress.analyzed': 'El catálogo produjo {count} hallazgo(s).',
  'audit.progress.done': 'Auditoría completada.',
  'audit.progress.cancelled': 'Auditoría cancelada.',

  'audit.error.title': 'La auditoría falló',
  'audit.error.bridge':
    'El runtime de la ventana no está disponible. Inicia la interfaz con "codixai ui".',
  'audit.error.generic': 'No se pudo completar la operación.',

  'origin.worktree': 'Árbol de trabajo (cambios sin confirmar)',
  'origin.staged': 'Área de preparación',
  'origin.base': 'Revisión base',

  'summary.title': 'Resumen',
  'summary.root': 'Repositorio',
  'summary.origin': 'Origen',
  'summary.base': 'Base',
  'summary.changedFiles': 'Archivos cambiados',
  'summary.findings': 'Hallazgos',
  'summary.failOn': 'Falla con',
  'summary.failed': 'Los hallazgos alcanzan el umbral de fallo.',
  'summary.passed': 'Ningún hallazgo alcanza el umbral de fallo.',
  'summary.known': 'Conocidos por la línea base',
  'summary.suppressed': 'Suprimidos por la configuración',
  'summary.excluded': 'Archivos excluidos',
  'summary.none': 'ninguno',

  'findings.title': 'Hallazgos',
  'findings.count': '{count} hallazgo(s)',
  'findings.empty': 'Sin hallazgos.',
  'findings.emptyHint':
    'La ausencia de hallazgos no demuestra que el cambio sea correcto.',
  'findings.noResults': 'Ningún hallazgo coincide con los filtros.',
  'findings.noResultsHint': 'Limpia los filtros para ver el documento completo.',

  'filters.title': 'Filtros',
  'filters.rule': 'Regla',
  'filters.severity': 'Severidad',
  'filters.epistemic': 'Estado epistémico',
  'filters.language': 'Lenguaje',
  'filters.path': 'Ruta',
  'filters.pathPlaceholder': 'Filtrar por ruta…',
  'filters.sort': 'Orden',
  'filters.all': 'Todos',
  'filters.clear': 'Limpiar filtros',
  'filters.sortPath': 'Ruta',
  'filters.sortSeverity': 'Severidad',
  'filters.sortRule': 'Regla',
  'filters.sortEpistemic': 'Estado epistémico',
  'filters.sortLanguage': 'Lenguaje',
  'filters.ascending': 'Ascendente',
  'filters.descending': 'Descendente',

  'table.severity': 'Severidad',
  'table.rule': 'Regla',
  'table.epistemic': 'Epistémico',
  'table.language': 'Lenguaje',
  'table.location': 'Ubicación',
  'table.message': 'Mensaje',

  'detail.title': 'Detalle del hallazgo',
  'detail.empty': 'Selecciona un hallazgo para leer su evidencia.',
  'detail.message': 'Mensaje',
  'detail.location': 'Ubicación',
  'detail.ruleVersion': 'Versión de la regla',
  'detail.evidence': 'Evidencia',
  'detail.evidenceKind': 'Tipo',
  'detail.evidenceRef': 'Referencia',
  'detail.hunk': 'Fragmento del diff',
  'detail.hunkEmpty': 'El conjunto de cambios no contiene el fragmento de esta evidencia.',
  'detail.support': 'Soporte',
  'detail.supportLevel': 'Nivel de soporte',
  'detail.method': 'Método',
  'detail.limitations': 'Limitaciones',
  'detail.recommendation': 'Recomendación',
  'detail.revision': 'Revisión',
  'detail.base': 'Base',
  'detail.head': 'Head',
  'detail.none': 'Ninguna declarada.',

  'severity.info': 'Información',
  'severity.warning': 'Advertencia',
  'severity.error': 'Error',

  'epistemic.observed': 'Observado',
  'epistemic.inferred': 'Inferido',
  'epistemic.unknown': 'Desconocido',

  'support.none': 'Ninguno',
  'support.syntactic': 'Sintáctico',
  'support.resolved': 'Resuelto',

  'export.title': 'Exportar',
  'export.json': 'Exportar JSON',
  'export.sarif': 'Exportar SARIF',
  'export.saveTitle': 'Guardar el informe de la auditoría',
  'export.saved': 'Informe escrito en {path}.',
  'export.cancelled': 'La exportación se canceló.',

  'footer.note':
    'Codixai lee el repositorio: nunca lo modifica, nunca ejecuta su código y nunca accede a la red.',

  'tab.audit': 'Auditoría',
  'tab.verify': 'Verificación',

  'verify.title': 'Verificación',
  'verify.repository': 'Raíz del repositorio',
  'verify.repositoryPlaceholder': '/ruta/al/repositorio',
  'verify.chooseRepository': 'Elegir…',
  'verify.chooseRepositoryTitle': 'Selecciona el repositorio que quieres verificar',
  'verify.base': 'Revisión base',
  'verify.basePlaceholder': 'main',
  'verify.claimsFile': 'Archivo de claims',
  'verify.claimsPlaceholder': '/ruta/a/claims.json',
  'verify.chooseClaims': 'Elegir…',
  'verify.chooseClaimsTitle': 'Selecciona el archivo de claims',
  'verify.reportFile': 'Archivo de informe de tests',
  'verify.reportPlaceholder': '/ruta/a/report.xml',
  'verify.chooseReport': 'Elegir…',
  'verify.chooseReportTitle': 'Selecciona el informe de tests',
  'verify.run': 'Ejecutar verificación',
  'verify.running': 'Verificando…',
  'verify.cancel': 'Cancelar',
  'verify.idle': 'Todavía no hay verificación.',
  'verify.idleHint':
    'Elige el repositorio, la revisión base, el archivo de claims y opcionalmente el informe de tests, y ejecuta la verificación.',

  'verify.error.noRoot': 'Debes especificar la raíz del repositorio.',
  'verify.error.noBase': 'Debes especificar la revisión base.',
  'verify.error.noClaims': 'Debes especificar el archivo de claims.',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.progress.validating': 'Validando la solicitud…',
  'verify.progress.collecting': 'Recopilando los cambios…',
  'verify.progress.collected': 'Se recopilaron {count} archivo(s) cambiado(s).',
  'verify.progress.evaluating': 'Evaluando los claims…',
  'verify.progress.evaluated': 'Se evaluaron {count} claim(s).',
  'verify.progress.done': 'Verificación completada.',
  'verify.progress.cancelled': 'Verificación cancelada.',

  'verify.error.title': 'La verificación falló',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.results': 'Resultados',
  'verify.resultsEmpty': 'Sin resultados.',
  'verify.claimId': 'ID',
  'verify.state': 'Estado',
  'verify.severity': 'Severidad',
  'verify.claim': 'Claim',
  'verify.paths': 'Rutas',
  'verify.message': 'Mensaje',
  'verify.evidence': 'Evidencia',
  'verify.limitations': 'Limitaciones',
  'verify.detail': 'Detalle',
  'verify.claim': 'Claim',

  'verify.stateVerified': 'Verificado',
  'verify.stateContradicted': 'Contradicho',
  'verify.stateUnresolved': 'No resuelto',

  'verify.claimsFile': 'Archivo de claims',
  'verify.claimsPlaceholder': '/ruta/a/claims.json',
  'verify.chooseClaims': 'Elegir…',
  'verify.chooseClaimsTitle': 'Selecciona el archivo de claims',
  'verify.reportFile': 'Archivo de informe de tests',
  'verify.reportPlaceholder': '/ruta/a/report.xml',
  'verify.chooseReport': 'Elegir…',
  'verify.chooseReportTitle': 'Selecciona el informe de tests',
  'verify.run': 'Ejecutar verificación',
  'verify.running': 'Verificando…',
  'verify.cancel': 'Cancelar',
  'verify.idle': 'Todavía no hay verificación.',
  'verify.idleHint': 'Elige el repositorio, la revisión base, el archivo de claims y opcionalmente el informe de tests, y ejecuta la verificación.',

  'verify.error.noRoot': 'Debes especificar la raíz del repositorio.',
  'verify.error.noBase': 'Debes especificar la revisión base.',
  'verify.error.noClaims': 'Debes especificar el archivo de claims.',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.progress.validating': 'Validando la solicitud…',
  'verify.progress.collecting': 'Recopilando los cambios…',
  'verify.progress.collected': 'Se recopilaron {count} archivo(s) cambiado(s).',
  'verify.progress.evaluating': 'Evaluando los claims…',
  'verify.progress.evaluated': 'Se evaluaron {count} claim(s).',
  'verify.progress.done': 'Verificación completada.',
  'verify.progress.cancelled': 'Verificación cancelada.',

  'verify.error.title': 'La verificación falló',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.results': 'Resultados',
  'verify.resultsEmpty': 'Sin resultados.',
  'verify.claimId': 'ID',
  'verify.state': 'Estado',
  'verify.severity': 'Severidad',
  'verify.claim': 'Claim',
  'verify.paths': 'Rutas',
  'verify.message': 'Mensaje',
  'verify.evidence': 'Evidencia',
  'verify.limitations': 'Limitaciones',
  'verify.detail': 'Detalle',
  'verify.claim': 'Claim',

  'verify.stateVerified': 'Verificado',
  'verify.stateContradicted': 'Contradicho',
  'verify.stateUnresolved': 'No resuelto',

  'verify.claimsFile': 'Archivo de claims',
  'verify.claimsPlaceholder': '/ruta/a/claims.json',
  'verify.chooseClaims': 'Elegir…',
  'verify.chooseClaimsTitle': 'Selecciona el archivo de claims',
  'verify.reportFile': 'Archivo de informe de tests',
  'verify.reportPlaceholder': '/ruta/a/report.xml',
  'verify.chooseReport': 'Elegir…',
  'verify.chooseReportTitle': 'Selecciona el informe de tests',
  'verify.run': 'Ejecutar verificación',
  'verify.running': 'Verificando…',
  'verify.cancel': 'Cancelar',
  'verify.idle': 'Todavía no hay verificación.',
  'verify.idleHint': 'Elige el repositorio, la revisión base, el archivo de claims y opcionalmente el informe de tests, y ejecuta la verificación.',

  'verify.error.noRoot': 'Debes especificar la raíz del repositorio.',
  'verify.error.noBase': 'Debes especificar la revisión base.',
  'verify.error.noClaims': 'Debes especificar el archivo de claims.',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.progress.validating': 'Validando la solicitud…',
  'verify.progress.collecting': 'Recopilando los cambios…',
  'verify.progress.collected': 'Se recopilaron {count} archivo(s) cambiado(s).',
  'verify.progress.evaluating': 'Evaluando los claims…',
  'verify.progress.evaluated': 'Se evaluaron {count} claim(s).',
  'verify.progress.done': 'Verificación completada.',
  'verify.progress.cancelled': 'Verificación cancelada.',

  'verify.error.title': 'La verificación falló',
  'verify.error.generic': 'No se pudo completar la operación.',

  'verify.results': 'Resultados',
  'verify.resultsEmpty': 'Sin resultados.',
  'verify.claimId': 'ID',
  'verify.state': 'Estado',
  'verify.severity': 'Severidad',
  'verify.claim': 'Claim',
  'verify.paths': 'Rutas',
  'verify.message': 'Mensaje',
  'verify.evidence': 'Evidencia',
  'verify.limitations': 'Limitaciones',
  'verify.detail': 'Detalle',
  'verify.claim': 'Claim',

  'verify.stateVerified': 'Verificado',
  'verify.stateContradicted': 'Contradicho',
  'verify.stateUnresolved': 'No resuelto',

  'tab.audit': 'Auditoría',
  'tab.verify': 'Verificación',

const dictionaries: Record<Locale, Record<string, string>> = { en, es }

// detectLocale returns the persisted locale, or the language of the system
// when the interface has never been used.
function detectLocale(): Locale {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'en' || stored === 'es') {
      return stored
    }
  } catch {
    // Storage can be unavailable; the language of the system is enough.
  }
  const language =
    typeof navigator !== 'undefined' ? navigator.language.toLowerCase() : 'en'
  return language.startsWith('es') ? 'es' : 'en'
}

class I18n {
  locale = $state<Locale>(detectLocale())

  // t returns the text of a key with the parameters interpolated.
  t(key: string, params?: Record<string, string | number>): string {
    const dictionary = dictionaries[this.locale]
    let text = dictionary[key] ?? dictionaries.en[key] ?? key
    if (params) {
      for (const [name, value] of Object.entries(params)) {
        text = text.replaceAll(`{${name}}`, String(value))
      }
    }
    return text
  }

  // setLocale changes the language, persists it and updates the document.
  setLocale(locale: Locale): void {
    this.locale = locale
    try {
      localStorage.setItem(STORAGE_KEY, locale)
    } catch {
      // The language still changes for this session.
    }
    if (typeof document !== 'undefined') {
      document.documentElement.lang = locale
    }
  }
}

export const i18n = new I18n()

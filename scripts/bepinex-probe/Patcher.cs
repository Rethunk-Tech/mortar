// A BepInEx preloader patcher for the matrix regress: it patches nothing and logs that BepInEx found it in patchers/.
using System.Collections.Generic;
using Mono.Cecil;

public static class MatrixPatcher
{
    public static IEnumerable<string> TargetDLLs
    {
        get { return new string[0]; }
    }

    public static void Initialize()
    {
        BepInEx.Logging.Logger.CreateLogSource("MatrixPatcher").LogInfo("matrix patcher initialized");
    }

    public static void Patch(AssemblyDefinition assembly)
    {
    }
}
